# caps-switch

<p align="center">
  <img src="assets/icon.png" width="128" alt="caps-switch icon">
</p>

<p align="center">
  <a href="https://github.com/dimzon/caps-switch/actions/workflows/ci.yml"><img src="https://github.com/dimzon/caps-switch/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/dimzon/caps-switch/actions/workflows/release.yml"><img src="https://github.com/dimzon/caps-switch/actions/workflows/release.yml/badge.svg" alt="Release"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="MIT"></a>
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
  | `Alt+Shift` | `SendInput` of `Alt+Shift` | Emulation; needs the system hotkey |
  | `Ctrl+Shift` | `SendInput` of `Ctrl+Shift` | Emulation; needs the system hotkey |

- **Tray icon** with a checkmark menu: pick a mode, or exit.
- **No dependencies.** One static `.exe`, pure Win32 through `syscall`.
- **Exit** from the tray menu.

## Build

Requires Go 1.21+.

The app icon is embedded into the executable with [`rsrc`](https://github.com/akavel/rsrc). The included script does it for you:

```powershell
./build.ps1
```

Or manually:

```sh
go run github.com/akavel/rsrc@latest -ico assets/icon.ico -o rsrc.syso -arch amd64
go build -trimpath -ldflags "-s -w -H windowsgui" -o caps-switch.exe .
```

At runtime the tray icon is loaded from `icon.ico` or `assets/icon.ico` next to the executable; otherwise the embedded/system icon is used.

Prebuilt binaries for every release are on the [Releases](https://github.com/dimzon/caps-switch/releases) page.

## Usage

1. Run `caps-switch.exe`.
2. Press **Caps Lock** — the layout flips.
3. Right- or left-click the tray icon to choose a mode or exit.
4. To autostart, put a shortcut to `caps-switch.exe` in `shell:startup`.

### Emulation modes (`Alt+Shift`, `Ctrl+Shift`)

These modes send the real chord with `SendInput`, so Windows must already have the matching switching hotkey assigned. **caps-switch never writes to the registry** — use the ready-made files in [`registry/`](registry/):

| File | Effect |
|------|--------|
| [`registry/alt-shift.reg`](registry/alt-shift.reg) | assign Alt+Shift (`LanguageHotkey`/`Hotkey` = `1`) |
| [`registry/ctrl-shift.reg`](registry/ctrl-shift.reg) | assign Ctrl+Shift (= `2`) |
| [`registry/restore-default.reg`](registry/restore-default.reg) | remove the assignment and restore defaults |

Double-click a file and confirm, or run `reg import registry\alt-shift.reg`. A sign-out may be needed for the hotkey to take effect. `direct` and `attach` do not need any hotkey at all.

## How it works

`caps-switch` installs a low-level keyboard hook (`WH_KEYBOARD_LL`). When Caps Lock is pressed it swallows the key and switches the input language of the foreground window using the selected backend. The layout list comes from `GetKeyboardLayoutList`; the target layout is tracked in-process so the toggle keeps working even in hosts (like `conhost`) where `GetKeyboardLayout` reports a stale handle.

## A note on Windows Terminal

Switching is instant in classic `conhost`. In **Windows Terminal** you may see a delay of up to a few seconds, because Windows Terminal is a WinUI/TSF app while `conhost` uses the legacy IMM32 input stack. That is a property of the terminal, not of the switcher. If it bothers you, use `conhost`, WezTerm, or Alacritty.

## License

MIT — see [LICENSE](LICENSE).
