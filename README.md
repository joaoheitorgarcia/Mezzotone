[![Version](https://img.shields.io/badge/Version-v1.1.1-blue.svg)](https://github.com/JoaoHeitorGarcia/Mezzotone/releases)
[![Powered by Bubble Tea](https://img.shields.io/badge/Powered_by-Bubble_Tea-7a4a8f)](https://github.com/charmbracelet/bubbletea)
[![Powered by Go](https://img.shields.io/badge/Powered_by-Go-7a4a8f)](https://github.com/golang/go)

# Mezzotone

Mezzotone is a terminal UI (TUI) app written in Go that converts images and GIFs into ASCII/Unicode art.

<img width="1865" height="1002" alt="image" src="https://github.com/user-attachments/assets/731ab90b-afbe-4bee-a0fb-36875029db84" />

## Features

- Convert `png`, `jpg`, `jpeg`, `bmp`, `webp`, `tiff`, and `gif`
- Multiple rune modes: `ASCII`, `UNICODE`, `DOTS`, `RECTANGLES`, `BARS`
- Optional colored rendering in terminal and exports
- Adjustable render settings (text size, font aspect, contrast, edge threshold, etc.)
- Export generated output to:
  - `.txt`
  - `.png`
  - `.gif`
- Clipboard copy support from the render view

## Install

### Run from source

Requirements:

- Go `1.25.6` or newer

```bash
git clone https://github.com/JoaoHeitorGarcia/Mezzotone.git
cd Mezzotone
go run .
```

### Use prebuilt binaries

Prebuilt binaries are available in the [`build`](./build) directory in this repository.

## CLI usage

```bash
mezzotone [flags]
```

Flags:

- `-debug`: enable debug logging to `logs.log`
- `-font-ttf <path>`: use a custom `.ttf` when exporting image/gif files

Example:

```bash
go run . -debug -font-ttf /path/to/font.ttf
```

## Quick workflow

1. Pick an image/GIF in the file picker.
2. Tune render settings in the options panel.
3. Press `enter` on confirm to render.
4. In render view:
   - `c` copy to clipboard
   - `t` export to `.txt`
   - `i` export to `.png`
   - `g` export to `.gif`

Exported files are written to your home directory with names like `Mezzotone_<uuid>.png`.

## Key controls

Global:

- `h`: toggle help
- `esc`: back (or press twice in file picker to quit)
- `ctrl+c`: quit

File picker:

- `j`/`k` or arrows: move
- `enter`/`right`: open directory or select file
- `left`/`backspace`: go to parent directory
- `pgup`/`pgdown`: jump

Render options:

- `j`/`k` or arrows: move
- `enter`: edit/confirm
- `space`: toggle boolean
- `left`/`right`: change enum values

Render view:

- Arrows: scroll
- `f`: toggle fullscreen
- `pgup`/`pgdown`: page scroll
- `shift+left`/`shift+right`: jump horizontal start/end

## Clipboard notes

Mezzotone uses `golang.design/x/clipboard` and falls back to system tools when available.

On Linux, install one of:

- `wl-copy`
- `xclip`
- `xsel`

## Build binaries

Use the build script:

```bash
bash tools/build.sh
```

Generated binaries are written to `build/` for Linux, macOS, and Windows targets.

## Notes

- GIF playback and GIF export are supported.
- Video conversion is not currently wired in the TUI workflow yet, maybe in the future.

## Examples

#### Image
Original
<img width="1920" height="645" alt="image" src="https://github.com/user-attachments/assets/a5a0325a-fb04-47a8-acd2-6f488a96db75" />
<img width="959" height="1003" alt="image" src="https://github.com/user-attachments/assets/1b0e38b3-8acc-4299-a227-f60d62e6029b" />

##

#### Gif
Original:

![test](https://github.com/user-attachments/assets/8a9e654b-354b-4aae-a3a2-0bd2a96a0ade)

Output:

![Mezzotone_b594ace5-b402-4d52-8786-d909c741bcb9](https://github.com/user-attachments/assets/11ee0c34-0724-4e2a-be84-a7fa48c68a9e)
![Mezzotone_8c87d3ae-26d1-46cd-bb07-1f164428aacb](https://github.com/user-attachments/assets/2896d241-b292-4308-8e85-1eed11361976)

