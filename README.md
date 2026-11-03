# caps-switch

<p align="center">
  <img src="assets/icon.png" width="128" alt="caps-switch icon">
</p>

<p align="center"><b>Single-key keyboard layout switching on Windows — press Caps Lock, flip the layout.</b></p>

Windows has no built-in way to switch the keyboard layout with a single key. **caps-switch** makes **Caps Lock** do exactly that (like `grp:caps_toggle` on Linux), with a tray icon and a live mode switcher.

## Features

- **Caps Lock = layout switch.** One keypress flips between layouts; Caps Lock itself is swallowed.
- **Three switching backends**, switchable at runtime from the tray:

  | Mode | How it switches | Notes |
  |------|-----------------|-------|
  | `direct` (default) | `SendMessage(WM_INPUTLANGCHANGEREQUEST)` | Fast; works in most apps |
  | `attach` | `AttachThreadInput` + `ActivateKeyboardLayout` | Bypasses the message queue |
  | `keys` | `SendInput` of `Ctrl+Shift` | Universal fallback; needs `LanguageHotkey=2` |

- **Tray icon** with a checkmark menu: pick a mode, or exit.
- **No dependencies.** One static `.exe`, pure Win32 through `syscall`.
- **Exit** with `Ctrl+Alt+Q` or the tray menu.

## Build

Requires Go 1.21+.

```sh
go build -ldflags "-H windowsgui" -o caps-switch.exe .
```

The tray icon is loaded from `icon.ico` or `assets/icon.ico` next to the executable; otherwise the system icon is used.

## Usage

1. Run `caps-switch.exe`.
2. Press **Caps Lock** — the layout flips.
3. Right- or left-click the tray icon to choose a mode or exit.
4. To autostart, put a shortcut to `caps-switch.exe` in `shell:startup`.

### The `keys` mode

`keys` emulates the `Ctrl+Shift` hotkey, so Windows must have that hotkey assigned:

```
HKCU\Keyboard Layout\Toggle
  LanguageHotkey = "2"   (and/or Hotkey = "2")
```

`direct` and `attach` do not need any hotkey.

## How it works

`caps-switch` installs a low-level keyboard hook (`WH_KEYBOARD_LL`). When Caps Lock is pressed it swallows the key and switches the input language of the foreground window using the selected backend. The layout list comes from `GetKeyboardLayoutList`; the target layout is tracked in-process so the toggle keeps working even in hosts (like `conhost`) where `GetKeyboardLayout` reports a stale handle.

## A note on Windows Terminal

Switching is instant in classic `conhost`. In **Windows Terminal** you may see a delay of up to a few seconds, because Windows Terminal is a WinUI/TSF app while `conhost` uses the legacy IMM32 input stack. That is a property of the terminal, not of the switcher. If it bothers you, use `conhost`, WezTerm, or Alacritty.

## License

MIT — see [LICENSE](LICENSE).
