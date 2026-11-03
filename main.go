// caps-switch — Caps Lock переключает раскладку + иконка в трее.
//
// Меню в трее: выбор режима (Direct / Attach / Alt+Shift / Ctrl+Shift) и выход.
//   direct     — SendMessage(WM_INPUTLANGCHANGEREQUEST) в активное окно
//   attach     — AttachThreadInput + ActivateKeyboardLayout (самый "прямой")
//   alt+shift  — эмуляция Alt+Shift через SendInput (хоткей задаётся в системе вручную)
//   ctrl+shift — эмуляция Ctrl+Shift через SendInput (хоткей задаётся в системе вручную)
//
// Тул не пишет в реестр.
//
// Сборка:
//   go build -ldflags "-H windowsgui" -o caps-switch.exe .
//
// Выход: только через меню в трее.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")

	procSetWindowsHookExW        = user32.NewProc("SetWindowsHookExW")
	procUnhookWindowsHookEx      = user32.NewProc("UnhookWindowsHookEx")
	procCallNextHookEx           = user32.NewProc("CallNextHookEx")
	procGetMessageW              = user32.NewProc("GetMessageW")
	procTranslateMessage         = user32.NewProc("TranslateMessage")
	procDispatchMessageW         = user32.NewProc("DispatchMessageW")
	procGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procGetKeyboardLayout        = user32.NewProc("GetKeyboardLayout")
	procGetKeyboardLayoutList    = user32.NewProc("GetKeyboardLayoutList")
	procPostMessageW             = user32.NewProc("PostMessageW")
	procSendInput                = user32.NewProc("SendInput")
	procAttachThreadInput        = user32.NewProc("AttachThreadInput")
	procActivateKeyboardLayout   = user32.NewProc("ActivateKeyboardLayout")
	procRegisterClassExW         = user32.NewProc("RegisterClassExW")
	procCreateWindowExW          = user32.NewProc("CreateWindowExW")
	procDefWindowProcW           = user32.NewProc("DefWindowProcW")
	procDestroyWindow            = user32.NewProc("DestroyWindow")
	procPostQuitMessage          = user32.NewProc("PostQuitMessage")
	procCreatePopupMenu          = user32.NewProc("CreatePopupMenu")
	procAppendMenuW              = user32.NewProc("AppendMenuW")
	procDestroyMenu              = user32.NewProc("DestroyMenu")
	procTrackPopupMenu           = user32.NewProc("TrackPopupMenu")
	procSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
	procGetCursorPos             = user32.NewProc("GetCursorPos")
	procLoadIconW                = user32.NewProc("LoadIconW")
	procLoadImageW               = user32.NewProc("LoadImageW")
	procSendMessageTimeoutW      = user32.NewProc("SendMessageTimeoutW")
	procPeekMessageW             = user32.NewProc("PeekMessageW")
	procShellNotifyIconW         = shell32.NewProc("Shell_NotifyIconW")
	procGetCurrentThreadId       = kernel32.NewProc("GetCurrentThreadId")
	procGetModuleHandleW         = kernel32.NewProc("GetModuleHandleW")
)

const (
	WH_KEYBOARD_LL            = 13
	WM_KEYDOWN                = 0x0100
	WM_KEYUP                  = 0x0101
	WM_SYSKEYDOWN             = 0x0104
	WM_SYSKEYUP               = 0x0105
	WM_INPUTLANGCHANGEREQUEST = 0x0050
	WM_COMMAND                = 0x0111
	WM_DESTROY                = 0x0002
	WM_NULL                   = 0x0000
	WM_APP                    = 0x8000
	WM_LBUTTONUP              = 0x0202
	WM_RBUTTONUP              = 0x0205
	VK_CAPITAL                = 0x14
	VK_LCONTROL               = 0xA2
	VK_LALT                   = 0xA4
	VK_LSHIFT                 = 0xA0
	INPUT_KEYBOARD            = 1
	KEYEVENTF_KEYUP           = 0x0002

	MF_STRING       = 0x0000
	MF_CHECKED      = 0x0008
	MF_SEPARATOR    = 0x0800
	TPM_RETURNCMD   = 0x0100
	TPM_RIGHTBUTTON = 0x0002

	SMTO_ABORTIFHUNG = 0x0002
	PM_NOREMOVE      = 0x0000

	IMAGE_ICON      = 0x0001
	LR_LOADFROMFILE = 0x0010
	LR_DEFAULTSIZE  = 0x0040

	NIF_MESSAGE = 0x00000001
	NIF_ICON    = 0x00000002
	NIF_TIP     = 0x00000004
	NIM_ADD     = 0x00000000
	NIM_MODIFY  = 0x00000001
	NIM_DELETE  = 0x00000002

	trayMsg = WM_APP + 1
)

const (
	modeDirect int32 = iota
	modeAttach
	modeAltShift
	modeCtrlShift
)

const (
	idModeDirect    = 1001
	idModeAttach    = 1002
	idModeAltShift  = 1003
	idModeCtrlShift = 1004
	idExit          = 1099
)

type KBDLLHOOKSTRUCT struct {
	VkCode      uint32
	ScanCode    uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

type KEYBDINPUT struct {
	WVk         uint16
	WScan       uint16
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

// INPUT = DWORD type + паддинг + union(32) на x64 => 40 байт.
type INPUT struct {
	Type  uint32
	_pad0 uint32
	Ki    KEYBDINPUT
	_pad1 [8]byte
}

type MSG struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

type POINT struct{ X, Y int32 }

type WNDCLASSEX struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  uintptr
	LpszClassName uintptr
	HIconSm       uintptr
}

type NOTIFYICONDATA struct {
	CbSize           uint32
	_pad0            uint32
	HWnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	_pad1            uint32
	HIcon            uintptr
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	UVersion         uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
	GuidItem         [16]byte
	HBalloonIcon     uintptr
}

var (
	switchReq    = make(chan struct{}, 1)
	modeCode     int32
	hwndTray     uintptr
	classNamePtr *uint16
	lastTarget   uintptr // последний HKL, который мы поставили (обход устаревшего GetKeyboardLayout в conhost)
)

// ---------- смена раскладки ----------

func nextLayout(tid uintptr) uintptr {
	n, _, _ := procGetKeyboardLayoutList.Call(0, 0)
	cnt := int(n)
	if cnt < 2 {
		return 0
	}
	buf := make([]uintptr, cnt)
	procGetKeyboardLayoutList.Call(uintptr(cnt), uintptr(unsafe.Pointer(&buf[0])))

	// База для "следующей". После первого переключения доверяем своему
	// lastTarget: в conhost GetKeyboardLayout(tid) может отдавать устаревший
	// HKL, из-за чего переключение залипало на одной раскладке.
	base := atomic.LoadUintptr(&lastTarget)
	if base == 0 {
		base, _, _ = procGetKeyboardLayout.Call(tid)
	}

	idx := -1
	for i, h := range buf {
		if h == base {
			idx = i
			break
		}
	}
	if idx < 0 {
		// сравниваем по младшему слову (языковой id), если HKL отличается старшим
		for i, h := range buf {
			if uint16(h) == uint16(base) {
				idx = i
				break
			}
		}
	}
	if idx < 0 {
		return buf[0]
	}
	return buf[(idx+1)%cnt]
}

func switchDirect() uintptr {
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return 0
	}
	tid, _, _ := procGetWindowThreadProcessId.Call(hwnd, 0)
	hklNext := nextLayout(tid)
	if hklNext == 0 {
		return 0
	}
	// SendMessage доставляется сразу (не ждёт очереди ввода), с таймаутом от зависаний.
	var res uintptr
	procSendMessageTimeoutW.Call(hwnd, WM_INPUTLANGCHANGEREQUEST, 0, hklNext, SMTO_ABORTIFHUNG, 100, uintptr(unsafe.Pointer(&res)))
	return hklNext
}

func switchAttach() uintptr {
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return 0
	}
	tid, _, _ := procGetWindowThreadProcessId.Call(hwnd, 0)
	if tid == 0 {
		return 0
	}
	hklNext := nextLayout(tid)
	if hklNext == 0 {
		return 0
	}
	cur, _, _ := procGetCurrentThreadId.Call()

	ok, _, _ := procAttachThreadInput.Call(cur, tid, 1)
	if ok == 0 {
		return 0
	}
	procActivateKeyboardLayout.Call(hklNext, 0)
	// Уведомляем окно тем же HKL (идемпотентно) — чтобы приложение точно обновило ввод.
	procPostMessageW.Call(hwnd, WM_INPUTLANGCHANGEREQUEST, 0, hklNext)
	procAttachThreadInput.Call(cur, tid, 0)
	return hklNext
}

// sendChord жмёт клавиши по порядку и отпускает в обратном.
func sendChord(keys ...uint16) {
	ins := make([]INPUT, 0, len(keys)*2)
	for _, k := range keys {
		ins = append(ins, INPUT{Type: INPUT_KEYBOARD, Ki: KEYBDINPUT{WVk: k}})
	}
	for i := len(keys) - 1; i >= 0; i-- {
		ins = append(ins, INPUT{Type: INPUT_KEYBOARD, Ki: KEYBDINPUT{WVk: keys[i], DwFlags: KEYEVENTF_KEYUP}})
	}
	procSendInput.Call(uintptr(len(ins)), uintptr(unsafe.Pointer(&ins[0])), unsafe.Sizeof(ins[0]))
}

func switchAltShift()  { sendChord(VK_LALT, VK_LSHIFT) }
func switchCtrlShift() { sendChord(VK_LCONTROL, VK_LSHIFT) }

func doSwitch() {
	var hkl uintptr
	switch atomic.LoadInt32(&modeCode) {
	case modeAttach:
		hkl = switchAttach()
	case modeAltShift:
		switchAltShift()
	case modeCtrlShift:
		switchCtrlShift()
	default:
		hkl = switchDirect()
	}
	if hkl != 0 {
		atomic.StoreUintptr(&lastTarget, hkl)
	}
}

func modeName(c int32) string {
	switch c {
	case modeAttach:
		return "attach"
	case modeAltShift:
		return "alt+shift"
	case modeCtrlShift:
		return "ctrl+shift"
	default:
		return "direct"
	}
}

// selectMode переключает режим и сбрасывает lastTarget. Реестр не трогаем.
func selectMode(m int32) {
	atomic.StoreInt32(&modeCode, m)
	atomic.StoreUintptr(&lastTarget, 0)
	updateTrayTip()
}

// ---------- трей ----------

func setTip(dst []uint16, s string) {
	u := syscall.StringToUTF16(s)
	n := copy(dst, u)
	if n < len(dst) {
		dst[n] = 0
	}
}

// loadAppIcon грузит icon.ico рядом с exe (или в assets/), иначе — системную.
func loadAppIcon() uintptr {
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		for _, rel := range []string{"icon.ico", filepath.Join("assets", "icon.ico")} {
			p := filepath.Join(dir, rel)
			if _, err := os.Stat(p); err != nil {
				continue
			}
			ptr, _ := syscall.UTF16PtrFromString(p)
			h, _, _ := procLoadImageW.Call(0, uintptr(unsafe.Pointer(ptr)), IMAGE_ICON, 0, 0, LR_LOADFROMFILE|LR_DEFAULTSIZE)
			if h != 0 {
				return h
			}
		}
	}
	h, _, _ := procLoadIconW.Call(0, 32512) // IDI_APPLICATION
	return h
}

func initTray() bool {
	hInst, _, _ := procGetModuleHandleW.Call(0)
	classNamePtr, _ = syscall.UTF16PtrFromString("CapsSwitchTrayWnd")

	var wc WNDCLASSEX
	wc.CbSize = uint32(unsafe.Sizeof(wc))
	wc.LpfnWndProc = syscall.NewCallback(wndProc)
	wc.HInstance = hInst
	wc.LpszClassName = uintptr(unsafe.Pointer(classNamePtr))
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))

	hwnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(classNamePtr)),
		0,
		0,
		0, 0, 0, 0,
		0, 0, hInst, 0,
	)
	if hwnd == 0 {
		return false
	}
	hwndTray = hwnd

	hIcon := loadAppIcon()

	var nid NOTIFYICONDATA
	nid.CbSize = uint32(unsafe.Sizeof(nid))
	nid.HWnd = hwnd
	nid.UID = 1
	nid.UFlags = NIF_MESSAGE | NIF_ICON | NIF_TIP
	nid.UCallbackMessage = trayMsg
	nid.HIcon = hIcon
	setTip(nid.SzTip[:], "caps-switch — режим: "+modeName(atomic.LoadInt32(&modeCode)))

	ok, _, _ := procShellNotifyIconW.Call(NIM_ADD, uintptr(unsafe.Pointer(&nid)))
	return ok != 0
}

func removeTray() {
	if hwndTray == 0 {
		return
	}
	var nid NOTIFYICONDATA
	nid.CbSize = uint32(unsafe.Sizeof(nid))
	nid.HWnd = hwndTray
	nid.UID = 1
	procShellNotifyIconW.Call(NIM_DELETE, uintptr(unsafe.Pointer(&nid)))
}

func updateTrayTip() {
	if hwndTray == 0 {
		return
	}
	var nid NOTIFYICONDATA
	nid.CbSize = uint32(unsafe.Sizeof(nid))
	nid.HWnd = hwndTray
	nid.UID = 1
	nid.UFlags = NIF_TIP
	setTip(nid.SzTip[:], "caps-switch — режим: "+modeName(atomic.LoadInt32(&modeCode)))
	procShellNotifyIconW.Call(NIM_MODIFY, uintptr(unsafe.Pointer(&nid)))
}

func appendMenu(h, flags, id uintptr, text string) {
	p, _ := syscall.UTF16PtrFromString(text)
	procAppendMenuW.Call(h, flags, id, uintptr(unsafe.Pointer(p)))
}

func check(b bool) uintptr {
	if b {
		return MF_CHECKED
	}
	return 0
}

func showMenu(hwnd uintptr) {
	hMenu, _, _ := procCreatePopupMenu.Call()
	if hMenu == 0 {
		return
	}
	defer procDestroyMenu.Call(hMenu)

	cur := atomic.LoadInt32(&modeCode)
	appendMenu(hMenu, MF_STRING|check(cur == modeDirect), idModeDirect, "Direct — WM_INPUTLANGCHANGEREQUEST")
	appendMenu(hMenu, MF_STRING|check(cur == modeAttach), idModeAttach, "Attach — ActivateKeyboardLayout")
	appendMenu(hMenu, MF_STRING|check(cur == modeAltShift), idModeAltShift, "Alt+Shift — эмуляция")
	appendMenu(hMenu, MF_STRING|check(cur == modeCtrlShift), idModeCtrlShift, "Ctrl+Shift — эмуляция")
	appendMenu(hMenu, MF_SEPARATOR, 0, "")
	appendMenu(hMenu, MF_STRING, idExit, "Выход")

	var pt POINT
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	procSetForegroundWindow.Call(hwnd)
	cmd, _, _ := procTrackPopupMenu.Call(
		hMenu,
		TPM_RETURNCMD|TPM_RIGHTBUTTON,
		uintptr(pt.X), uintptr(pt.Y),
		0, hwnd, 0,
	)
	procPostMessageW.Call(hwnd, WM_NULL, 0, 0)
	if cmd != 0 {
		handleCommand(hwnd, int(cmd))
	}
}

func handleCommand(hwnd uintptr, id int) {
	switch id {
	case idModeDirect:
		selectMode(modeDirect)
	case idModeAttach:
		selectMode(modeAttach)
	case idModeAltShift:
		selectMode(modeAltShift)
	case idModeCtrlShift:
		selectMode(modeCtrlShift)
	case idExit:
		procDestroyWindow.Call(hwnd)
	}
}

func wndProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	switch uint32(msg) {
	case trayMsg:
		m := uint32(lParam) & 0xFFFF
		if m == WM_RBUTTONUP || m == WM_LBUTTONUP {
			showMenu(hwnd)
		}
		return 0
	case WM_COMMAND:
		handleCommand(hwnd, int(uint32(wParam)&0xFFFF))
		return 0
	case WM_DESTROY:
		removeTray()
		procPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, msg, wParam, lParam)
	return r
}

// ---------- хук ----------

func hookProc(nCode uintptr, wParam uintptr, lParam unsafe.Pointer) uintptr {
	if int32(nCode) == 0 {
		kb := (*KBDLLHOOKSTRUCT)(lParam)
		if kb.VkCode == VK_CAPITAL {
			switch wParam {
			case WM_KEYDOWN, WM_SYSKEYDOWN:
				select {
				case switchReq <- struct{}{}:
				default:
				}
				return 1 // съедаем Caps
			case WM_KEYUP, WM_SYSKEYUP:
				return 1
			}
		}
	}
	r, _, _ := procCallNextHookEx.Call(0, nCode, wParam, uintptr(lParam))
	return r
}

func main() {
	modeFlag := flag.String("mode", "direct", "direct | attach | alt+shift | ctrl+shift")
	flag.Parse()
	switch *modeFlag {
	case "attach":
		selectMode(modeAttach)
	case "alt+shift", "altshift":
		selectMode(modeAltShift)
	case "ctrl+shift", "ctrlshift":
		selectMode(modeCtrlShift)
	default:
		selectMode(modeDirect)
	}

	runtime.LockOSThread()

	hook, _, err := procSetWindowsHookExW.Call(
		WH_KEYBOARD_LL,
		syscall.NewCallback(hookProc),
		0,
		0,
	)
	if hook == 0 {
		fmt.Println("Не удалось установить хук:", err)
		return
	}
	defer procUnhookWindowsHookEx.Call(hook)

	if initTray() {
		defer removeTray()
	} else {
		fmt.Println("Не удалось создать иконку в трее (работаю без неё).")
	}

	go func() {
		// AttachThreadInput требует у вызывающего потока очередь сообщений — создаём её.
		runtime.LockOSThread()
		var m MSG
		procPeekMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0, PM_NOREMOVE)
		for range switchReq {
			doSwitch()
		}
	}()

	fmt.Printf("caps-switch запущен (mode=%s). Caps = смена раскладки. Выход — меню в трее.\n",
		modeName(atomic.LoadInt32(&modeCode)))

	var msg MSG
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}
