package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"final-ride/internal/finalride"

	"github.com/atotto/clipboard"
	"github.com/sqweek/dialog"

	"golang.org/x/exp/shiny/materialdesign/icons"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// ========= Theme System =========

type ThemeColors struct {
	Bg, Sidebar, Surface, Primary, Text, TextLight, Border, Success, Error, TerminalBg, TerminalText color.NRGBA
}

var (
	LightTheme = ThemeColors{
		Bg:           color.NRGBA{R: 255, G: 255, B: 255, A: 255},
		Sidebar:      color.NRGBA{R: 248, G: 249, B: 250, A: 255},
		Surface:      color.NRGBA{R: 255, G: 255, B: 255, A: 255},
		Primary:      color.NRGBA{R: 26, G: 115, B: 232, A: 255},
		Text:         color.NRGBA{R: 60, G: 64, B: 67, A: 255},
		TextLight:    color.NRGBA{R: 95, G: 99, B: 104, A: 255},
		Border:       color.NRGBA{R: 218, G: 220, B: 224, A: 255},
		Success:      color.NRGBA{R: 24, G: 128, B: 56, A: 255},
		Error:        color.NRGBA{R: 217, G: 48, B: 37, A: 255},
		TerminalBg:   color.NRGBA{R: 30, G: 30, B: 30, A: 255},
		TerminalText: color.NRGBA{R: 200, G: 200, B: 200, A: 255},
	}

	DarkTheme = ThemeColors{
		Bg:           color.NRGBA{R: 32, G: 33, B: 36, A: 255},
		Sidebar:      color.NRGBA{R: 41, G: 42, B: 45, A: 255},
		Surface:      color.NRGBA{R: 48, G: 49, B: 52, A: 255},
		Primary:      color.NRGBA{R: 138, G: 180, B: 248, A: 255},
		Text:         color.NRGBA{R: 232, G: 234, B: 237, A: 255},
		TextLight:    color.NRGBA{R: 154, G: 160, B: 166, A: 255},
		Border:       color.NRGBA{R: 95, G: 99, B: 104, A: 255},
		Success:      color.NRGBA{R: 129, G: 201, B: 149, A: 255},
		Error:        color.NRGBA{R: 242, G: 139, B: 130, A: 255},
		TerminalBg:   color.NRGBA{R: 20, G: 20, B: 20, A: 255},
		TerminalText: color.NRGBA{R: 0, G: 255, B: 0, A: 255}, // Hackery green
	}

	CurrentTheme = LightTheme
)

// Icons
var (
	icMenu, icTheme, icUpload, icDownload, icSettings, icInfo, icClose, icCheck, icFolder *widget.Icon
)

// AppState holds the application state
type AppState struct {
	mu sync.Mutex

	// UI State
	currentTab     int // 0=Upload, 1=Download, 2=Settings
	isSidebarOpen  bool
	isDarkMode     bool
	filePath       string
	
	// Settings
	downloadDir    string
	encryptDefault bool

	metadataCID    string
	encryptFile    bool
	isProcessing   bool
	progress       float32
	status         string
	logs           []string
	resultCID      string
	speed          string

	// Connectivity
	isOnline bool
	lastPing time.Time

	// Stats
	startTime time.Time
}

// UI holds UI components
type UI struct {
	theme *material.Theme

	// Header
	menuBtn  widget.Clickable
	themeBtn widget.Clickable

	// Sidebar
	navUpload   widget.Clickable
	navDownload widget.Clickable
	navSettings widget.Clickable

	// Upload
	selectFileBtn widget.Clickable
	encryptCheck  widget.Bool
	uploadBtn     widget.Clickable

	// Download
	cidEditor   widget.Editor
	downloadBtn widget.Clickable

	// Settings
	settingsDownloadDirBtn widget.Clickable
	settingsEncryptCheck   widget.Bool
	settingsThemeSwitch    widget.Bool
	settingsSaveBtn        widget.Clickable
	settingsDownloadDirEd  widget.Editor

	// Common
	copyResultBtn widget.Clickable
	logsList      widget.List

	// File path input
	filePathEditor widget.Editor
}

var (
	config   *finalride.Config
	configMu sync.Mutex // Protects config
	appState *AppState
	ui       *UI
	window   *app.Window
)

func init() {
	// Load Icons
	icMenu, _ = widget.NewIcon(icons.NavigationMenu)
	icTheme, _ = widget.NewIcon(icons.ImageBrightness6) // Brightness/Theme toggle
	icUpload, _ = widget.NewIcon(icons.FileFileUpload)
	icDownload, _ = widget.NewIcon(icons.FileFileDownload)
	icSettings, _ = widget.NewIcon(icons.ActionSettings)
	icInfo, _ = widget.NewIcon(icons.ActionInfo)
	icClose, _ = widget.NewIcon(icons.NavigationClose)
	icCheck, _ = widget.NewIcon(icons.ActionCheckCircle)
	icFolder, _ = widget.NewIcon(icons.FileFolder)
}

func main() {
	config, _ = finalride.LoadConfig("config.yaml")
	
	// Default values if config missing
	if config.DownloadDir == "" {
		wd, _ := os.Getwd()
		config.DownloadDir = wd
	}

	appState = &AppState{
		encryptFile:    config.EncryptDefault,
		downloadDir:    config.DownloadDir,
		encryptDefault: config.EncryptDefault,
		logs:           make([]string, 0),
		isOnline:       false,
		isSidebarOpen:  true, // Default open
		isDarkMode:     config.Theme == "dark",
	}

	ui = &UI{}
	ui.theme = material.NewTheme()
	
	ui.encryptCheck.Value = appState.encryptFile
	ui.settingsEncryptCheck.Value = appState.encryptDefault
	ui.settingsThemeSwitch.Value = appState.isDarkMode
	ui.settingsDownloadDirEd.SetText(appState.downloadDir)
	
	ui.logsList.List.Axis = layout.Vertical
	ui.cidEditor.SingleLine = true
	ui.filePathEditor.SingleLine = true
	ui.settingsDownloadDirEd.SingleLine = true

	go func() {
		window = new(app.Window)
		window.Option(
			app.Title("Final Ride"),
			app.Size(unit.Dp(1000), unit.Dp(750)),
			app.MinSize(unit.Dp(600), unit.Dp(500)),
		)

		go startPingLoop()

		if err := run(window); err != nil {
			fmt.Println("Error:", err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func startPingLoop() {
	ticker := time.NewTicker(5 * time.Second)
	check := func() {
		client := http.Client{Timeout: 2 * time.Second}
		configMu.Lock()
		apiURL := config.SwarmAPI
		configMu.Unlock()
		resp, err := client.Get(apiURL)

		appState.mu.Lock()
		if err == nil && resp.StatusCode < 500 {
			appState.isOnline = true
		} else {
			appState.isOnline = false
		}
		appState.lastPing = time.Now()
		appState.mu.Unlock()

		if window != nil {
			window.Invalidate()
		}
	}
	check()
	for range ticker.C {
		check()
	}
}

func run(window *app.Window) error {
	var ops op.Ops

	for {
		switch e := window.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			drawUI(gtx)
			e.Frame(gtx.Ops)
		}
	}
}

func drawUI(gtx layout.Context) layout.Dimensions {
	appState.mu.Lock()
	darkMode := appState.isDarkMode
	sidebarOpen := appState.isSidebarOpen
	appState.mu.Unlock()

	if darkMode {
		CurrentTheme = DarkTheme
	} else {
		CurrentTheme = LightTheme
	}

	paint.Fill(gtx.Ops, CurrentTheme.Bg)

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// App Bar / Header
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return drawHeader(gtx)
		}),
		// Content Body
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				// Sidebar (Animated/Toggleable)
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if !sidebarOpen {
						return layout.Dimensions{}
					}
					return drawSidebar(gtx)
				}),
				// Divider (only if sidebar open)
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if !sidebarOpen {
						return layout.Dimensions{}
					}
					rect := image.Rectangle{Max: image.Point{X: gtx.Dp(unit.Dp(1)), Y: gtx.Constraints.Max.Y}}
					paint.FillShape(gtx.Ops, CurrentTheme.Border, clip.Rect(rect).Op())
					return layout.Dimensions{Size: rect.Max}
				}),
				// Main Content
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(24)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return drawContent(gtx)
					})
				}),
			)
		}),
	)
}

func drawHeader(gtx layout.Context) layout.Dimensions {
	// Header Background
	rect := image.Rectangle{Max: image.Point{X: gtx.Constraints.Max.X, Y: gtx.Dp(unit.Dp(60))}}
	paint.FillShape(gtx.Ops, CurrentTheme.Surface, clip.Rect(rect).Op())

	// Border Bottom
	borderRect := image.Rectangle{Min: image.Point{Y: rect.Max.Y - 1}, Max: rect.Max}
	paint.FillShape(gtx.Ops, CurrentTheme.Border, clip.Rect(borderRect).Op())

	return layout.Dimensions{Size: rect.Max} // Placeholder dimensions
		// Actually layout elements on top
		layout.Inset{Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				// Menu Button
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if ui.menuBtn.Clicked(gtx) {
						appState.mu.Lock()
						appState.isSidebarOpen = !appState.isSidebarOpen
						appState.mu.Unlock()
						window.Invalidate()
					}
					btn := material.IconButton(ui.theme, &ui.menuBtn, icMenu, "Toggle Menu")
					btn.Color = CurrentTheme.Text
					btn.Inset = layout.UniformInset(unit.Dp(12))
					return btn.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Spacer{Width: unit.Dp(16)}.Layout(gtx)
				}),
				// Logo Text (Header) - Remove or Keep? User said "add title name app on above sidebar".
				// I will keep it but maybe simplify or match sidebar?
				// "all font for text exept icon using montserrat font"
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					l := material.H6(ui.theme, "FINAL RIDE")
					l.Color = CurrentTheme.Text
					l.Font.Weight = font.Bold
					l.Font.Typeface = "Montserrat"
					return l.Layout(gtx)
				}),
				// Theme Toggle
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if ui.themeBtn.Clicked(gtx) {
						appState.mu.Lock()
						appState.isDarkMode = !appState.isDarkMode
						if appState.isDarkMode {
							config.Theme = "dark"
						} else {
							config.Theme = "light"
						}
						finalride.SaveConfig("config.yaml", config)
						appState.mu.Unlock()
						window.Invalidate()
					}
					btn := material.IconButton(ui.theme, &ui.themeBtn, icTheme, "Toggle Theme")
					btn.Color = CurrentTheme.Text
					return btn.Layout(gtx)
				}),
			)
		})
	return layout.Dimensions{Size: rect.Max}
}

func drawSidebar(gtx layout.Context) layout.Dimensions {
	paint.FillShape(gtx.Ops, CurrentTheme.Sidebar, clip.Rect{Max: gtx.Constraints.Max}.Op())
	width := unit.Dp(240) // Fixed width

	return layout.Stack{}.Layout(gtx,
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Dp(width)
			gtx.Constraints.Max.X = gtx.Dp(width)
			gtx.Constraints.Min.Y = gtx.Constraints.Max.Y // Force full height
			return layout.Inset{Top: unit.Dp(24), Bottom: unit.Dp(24), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						// App Title
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							l := material.H6(ui.theme, "FINAL RIDE")
							l.Color = CurrentTheme.Text
							l.Font.Weight = font.Bold
							l.Font.Typeface = "Montserrat"
							return layout.Inset{Bottom: unit.Dp(24), Left: unit.Dp(12)}.Layout(gtx, l.Layout)
						}),
						
						// Nav Buttons
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return drawNavButton(gtx, &ui.navUpload, "New Upload", 0, icUpload)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Spacer{Height: unit.Dp(8)}.Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return drawNavButton(gtx, &ui.navDownload, "Download", 1, icDownload)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Spacer{Height: unit.Dp(8)}.Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return drawNavButton(gtx, &ui.navSettings, "Settings", 2, icSettings)
						}),

						// Spacer
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return layout.Dimensions{}
						}),

						// Theme Toggle (Sidebar)
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							// Auto-Save: Theme
							if ui.settingsThemeSwitch.Value != appState.isDarkMode {
								appState.mu.Lock()
								appState.isDarkMode = ui.settingsThemeSwitch.Value
								
								// Non-blocking config save
								go func(isDark bool) {
									configMu.Lock()
									if isDark {
										config.Theme = "dark"
									} else {
										config.Theme = "light"
									}
									if err := finalride.SaveConfig("config.yaml", config); err != nil {
										addLog("Error saving theme: " + err.Error())
									}
									configMu.Unlock()
								}(appState.isDarkMode)
								
								appState.mu.Unlock()
								// Invalidate from outside the lock to avoid potential (though unlikely here) issues
								// but mostly to ensure UI redraws immediately.
								window.Invalidate()
							}
							
							sw := material.Switch(ui.theme, &ui.settingsThemeSwitch, "Dark Mode")
							sw.Color.Enabled = CurrentTheme.Primary
							sw.Color.Disabled = CurrentTheme.TextLight
							sw.Color.Track = color.NRGBA{A: 50}

							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return sw.Layout(gtx)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Spacer{Width: unit.Dp(8)}.Layout(gtx)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									l := material.Body2(ui.theme, "Dark Mode")
									l.Color = CurrentTheme.Text
									return l.Layout(gtx)
								}),
							)
						}),

						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Spacer{Height: unit.Dp(16)}.Layout(gtx)
						}),
						
						// Status
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return drawStatusIndicator(gtx)
						}),
					)
				},
			)
		}),
	)
}

func drawNavButton(gtx layout.Context, btn *widget.Clickable, label string, index int, icon *widget.Icon) layout.Dimensions {
	if btn.Clicked(gtx) {
		appState.mu.Lock()
		appState.currentTab = index
		appState.mu.Unlock()
		window.Invalidate()
	}

	appState.mu.Lock()
	selected := appState.currentTab == index
	appState.mu.Unlock()

	bgColor := color.NRGBA{A: 0}
	if selected {
		bgColor = CurrentTheme.Primary
		bgColor.A = 30 // Transparent tint
	}

	textColor := CurrentTheme.Text
	if selected {
		textColor = CurrentTheme.Primary
	}

	return btn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Stack{}.Layout(gtx,
			layout.Expanded(func(gtx layout.Context) layout.Dimensions {
				rr := gtx.Dp(unit.Dp(24))
				paint.FillShape(gtx.Ops, bgColor, clip.RRect{
					Rect: image.Rectangle{Max: gtx.Constraints.Min},
					NE: rr, NW: rr, SE: rr, SW: rr,
				}.Op(gtx.Ops))
				return layout.Dimensions{Size: gtx.Constraints.Min}
			}),
			layout.Stacked(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(12), Bottom: unit.Dp(12), Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								if icon != nil {
									return icon.Layout(gtx, textColor)
								}
								return layout.Dimensions{}
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Spacer{Width: unit.Dp(12)}.Layout(gtx)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								l := material.Body1(ui.theme, label)
								l.Color = textColor
								l.Font.Weight = font.Medium
								l.Font.Typeface = "Montserrat"
								return l.Layout(gtx)
							}),
						)
					},
				)
			}),
		)
	})
}

func drawStatusIndicator(gtx layout.Context) layout.Dimensions {
	appState.mu.Lock()
	isOnline := appState.isOnline
	appState.mu.Unlock()

	statusColor := CurrentTheme.Error
	statusText := "Server Offline"
	if isOnline {
		statusColor = CurrentTheme.Success
		statusText = "Server Online"
	}

	return widget.Border{Color: CurrentTheme.Border, CornerRadius: unit.Dp(4), Width: unit.Dp(1)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical, Alignment: layout.Start}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					l := material.Caption(ui.theme, "SERVER STATUS")
					l.Color = CurrentTheme.TextLight
					l.Font.Weight = font.Bold
					l.Font.Typeface = "Montserrat"
					return layout.Inset{Bottom: unit.Dp(4)}.Layout(gtx, l.Layout)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							size := gtx.Dp(unit.Dp(8))
							rect := image.Rectangle{Max: image.Point{X: size, Y: size}}
							paint.FillShape(gtx.Ops, statusColor, clip.Ellipse(rect).Op(gtx.Ops))
							return layout.Dimensions{Size: rect.Max}
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Spacer{Width: unit.Dp(8)}.Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							l := material.Body2(ui.theme, statusText) // body2 for visibility
							l.Color = CurrentTheme.Text
							l.Font.Typeface = "Montserrat"
							return l.Layout(gtx)
						}),
					)
				}),
			)
		})
	})
}

func drawContent(gtx layout.Context) layout.Dimensions {
	appState.mu.Lock()
	tab := appState.currentTab
	appState.mu.Unlock()

	if tab == 0 {
		return drawUploadTab(gtx)
	} else if tab == 1 {
		return drawDownloadTab(gtx)
	}
	return drawSettingsTab(gtx)
}

func drawUploadTab(gtx layout.Context) layout.Dimensions {
	// Browse logic
	if ui.selectFileBtn.Clicked(gtx) {
		go func() {
			filename, err := dialog.File().Title("Select file to upload").Load()
			if err == nil {
				ui.filePathEditor.SetText(filename)
				appState.mu.Lock()
				appState.filePath = filename
				appState.mu.Unlock()
				window.Invalidate()
			}
		}()
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			l := material.H6(ui.theme, "Upload File")
			l.Color = CurrentTheme.Text
			l.Font.Weight = font.Bold
			l.Font.Typeface = "Montserrat"
			return layout.Inset{Bottom: unit.Dp(20)}.Layout(gtx, l.Layout)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return drawCard(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								ed := material.Editor(ui.theme, &ui.filePathEditor, "Select a file...")
								ed.Color = CurrentTheme.Text
								ed.HintColor = CurrentTheme.TextLight
								ed.Font.Typeface = "Montserrat"
								border := widget.Border{Color: CurrentTheme.Border, CornerRadius: unit.Dp(4), Width: unit.Dp(1)}
								return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return layout.Inset{Top: unit.Dp(12), Bottom: unit.Dp(12), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, ed.Layout)
								})
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Spacer{Width: unit.Dp(12)}.Layout(gtx)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								btn := material.Button(ui.theme, &ui.selectFileBtn, "Browse")
								btn.Background = CurrentTheme.Primary
								btn.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
								return btn.Layout(gtx)
							}),
						)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Spacer{Height: unit.Dp(16)}.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						// Restore Encryption UI
						appState.mu.Lock()
						appState.encryptFile = ui.encryptCheck.Value
						appState.mu.Unlock()
						cb := material.CheckBox(ui.theme, &ui.encryptCheck, "Encrypt file (AES-256-GCM)")
						cb.Color = CurrentTheme.Text
						cb.IconColor = CurrentTheme.Primary
						cb.Font.Typeface = "Montserrat"
						return cb.Layout(gtx)
					}),
				)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Spacer{Height: unit.Dp(24)}.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return drawPrimaryActionBtn(gtx, &ui.uploadBtn, "Start Upload", func() {
				filePath := ui.filePathEditor.Text()
				if filePath != "" {
					go performUpload(filePath)
				}
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Spacer{Height: unit.Dp(24)}.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return drawProgressSection(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Spacer{Height: unit.Dp(24)}.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return drawResultSection(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Spacer{Height: unit.Dp(24)}.Layout(gtx)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return drawTerminal(gtx)
		}),
	)
}

func drawDownloadTab(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			l := material.H6(ui.theme, "Download File")
			l.Color = CurrentTheme.Text
			l.Font.Weight = font.Bold
			l.Font.Typeface = "Montserrat"
			return layout.Inset{Bottom: unit.Dp(20)}.Layout(gtx, l.Layout)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return drawCard(gtx, func(gtx layout.Context) layout.Dimensions {
				ed := material.Editor(ui.theme, &ui.cidEditor, "Paste Metadata CID here...")
				ed.Color = CurrentTheme.Text
				ed.HintColor = CurrentTheme.TextLight
				ed.Font.Typeface = "Montserrat"
				border := widget.Border{Color: CurrentTheme.Border, CornerRadius: unit.Dp(4), Width: unit.Dp(1)}
				return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: unit.Dp(12), Bottom: unit.Dp(12), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, ed.Layout)
				})
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Spacer{Height: unit.Dp(24)}.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return drawPrimaryActionBtn(gtx, &ui.downloadBtn, "Download", func() {
				cid := ui.cidEditor.Text()
				if cid != "" {
					go performDownload(cid)
				}
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Spacer{Height: unit.Dp(24)}.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return drawProgressSection(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Spacer{Height: unit.Dp(24)}.Layout(gtx)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return drawTerminal(gtx)
		}),
	)
}

func drawSettingsTab(gtx layout.Context) layout.Dimensions {
	// Handle Browse
	if ui.settingsDownloadDirBtn.Clicked(gtx) {
		go func() {
			dir, err := dialog.Directory().Title("Select Download Directory").Browse()
			if err == nil {
				ui.settingsDownloadDirEd.SetText(dir)
				appState.mu.Lock()
				appState.downloadDir = dir
				
				// Auto-Save Directory (Non-blocking)
				go func(newDir string) {
					configMu.Lock()
					config.DownloadDir = newDir
					if err := finalride.SaveConfig("config.yaml", config); err != nil {
						addLog("Error saving config: " + err.Error())
					} else {
						addLog("Directory setting updated")
					}
					configMu.Unlock()
				}(dir)

				appState.mu.Unlock()
				window.Invalidate()
			}
		}()
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			l := material.H6(ui.theme, "Settings")
			l.Color = CurrentTheme.Text
			l.Font.Weight = font.Bold
			l.Font.Typeface = "Montserrat"
			return layout.Inset{Bottom: unit.Dp(20)}.Layout(gtx, l.Layout)
		}),
		// Download Dir
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return drawCard(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						l := material.Body1(ui.theme, "Default Download Directory")
						l.Color = CurrentTheme.Text
						l.Font.Weight = font.Bold
						l.Font.Typeface = "Montserrat"
						return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, l.Layout)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								ed := material.Editor(ui.theme, &ui.settingsDownloadDirEd, "Select directory...")
								ed.Color = CurrentTheme.Text
								ed.HintColor = CurrentTheme.TextLight
								ed.Font.Typeface = "Montserrat"
								border := widget.Border{Color: CurrentTheme.Border, CornerRadius: unit.Dp(4), Width: unit.Dp(1)}
								return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return layout.Inset{Top: unit.Dp(12), Bottom: unit.Dp(12), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, ed.Layout)
								})
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Spacer{Width: unit.Dp(12)}.Layout(gtx)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								btn := material.IconButton(ui.theme, &ui.settingsDownloadDirBtn, icFolder, "Browse")
								btn.Color = CurrentTheme.Text
								btn.Inset = layout.UniformInset(unit.Dp(12))
								return btn.Layout(gtx)
							}),
						)
					}),
				)
			})
		}),

	)
}

func drawTerminal(gtx layout.Context) layout.Dimensions {
	// Frame styling
	border := widget.Border{Color: CurrentTheme.Border, CornerRadius: unit.Dp(6), Width: unit.Dp(1)}
	
	// Inner Terminal Background
	return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Stack{}.Layout(gtx,
			layout.Expanded(func(gtx layout.Context) layout.Dimensions {
				paint.FillShape(gtx.Ops, CurrentTheme.TerminalBg, clip.RRect{
					Rect: image.Rectangle{Max: gtx.Constraints.Min},
					NE: gtx.Dp(unit.Dp(6)), NW: gtx.Dp(unit.Dp(6)), SE: gtx.Dp(unit.Dp(6)), SW: gtx.Dp(unit.Dp(6)),
				}.Op(gtx.Ops))
				return layout.Dimensions{Size: gtx.Constraints.Min}
			}),
			layout.Stacked(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(12), Bottom: unit.Dp(12), Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							// Terminal Header
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										l := material.Body2(ui.theme, ">_ TERMINAL OUTPUT")
										l.Color = CurrentTheme.TerminalText
										l.Font.Weight = font.Bold
										l.Font.Typeface = "Montserrat" // Montserrat for header
										return l.Layout(gtx)
									}),
									layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }),
									// Status in Terminal?
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										appState.mu.Lock()
										status := appState.status
										appState.mu.Unlock()
										l := material.Caption(ui.theme, status)
										l.Color = CurrentTheme.TerminalText
										l.Font.Typeface = "Montserrat"
										return l.Layout(gtx)
									}),
								)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Spacer{Height: unit.Dp(8)}.Layout(gtx)
							}),
							// Logs List - Responsive Flexed
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								appState.mu.Lock()
								logs := make([]string, len(appState.logs))
								copy(logs, appState.logs)
								appState.mu.Unlock()

								return material.List(ui.theme, &ui.logsList).Layout(gtx, len(logs), func(gtx layout.Context, i int) layout.Dimensions {
									l := material.Body2(ui.theme, logs[i])
									l.Color = CurrentTheme.TerminalText
									l.Font.Typeface = "Montserrat"
									return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2)}.Layout(gtx, l.Layout)
								})
							}),
						)
					},
				)
			}),
		)
	})
}

// Reuse existing Card/Button helpers but updated to use CurrentTheme
func drawResultSection(gtx layout.Context) layout.Dimensions {
	appState.mu.Lock()
	resultCID := appState.resultCID
	appState.mu.Unlock()

	// Handle Copy
	if ui.copyResultBtn.Clicked(gtx) {
		clipboard.WriteAll(resultCID)
	}

	if resultCID == "" {
		return layout.Dimensions{}
	}

	return drawCard(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := material.Body2(ui.theme, "Upload Complete. Metadata CID:")
				l.Color = CurrentTheme.Success
				l.Font.Weight = font.Bold
				l.Font.Typeface = "Montserrat"
				return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, l.Layout)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						l := material.Body1(ui.theme, resultCID)
						l.Color = CurrentTheme.Text
						l.Font.Typeface = "Montserrat"
						return l.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Spacer{Width: unit.Dp(16)}.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						btn := material.Button(ui.theme, &ui.copyResultBtn, "Copy")
						btn.Background = CurrentTheme.Surface
						btn.Color = CurrentTheme.Primary
						btn.Inset = layout.UniformInset(unit.Dp(10))
						return btn.Layout(gtx)
					}),
				)
			}),
		)
	})
}

func drawProgressSection(gtx layout.Context) layout.Dimensions {
	appState.mu.Lock()
	progress := appState.progress
	isProcessing := appState.isProcessing
	status := appState.status
	speed := appState.speed
	appState.mu.Unlock()

	if !isProcessing && progress <= 0 {
		return layout.Dimensions{}
	}

	return drawCard(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						l := material.Body2(ui.theme, status)
						l.Color = CurrentTheme.Text
						l.Font.Weight = font.Bold
						l.Font.Typeface = "Montserrat"
						return l.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if speed == "" || !isProcessing {
							return layout.Dimensions{}
						}
						l := material.Caption(ui.theme, speed)
						l.Color = CurrentTheme.Primary
						l.Font.Weight = font.Bold
						l.Font.Typeface = "Montserrat"
						return l.Layout(gtx)
					}),
				)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Spacer{Height: unit.Dp(12)}.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				pb := material.ProgressBar(ui.theme, progress)
				pb.Color = CurrentTheme.Primary
				pb.TrackColor = color.NRGBA{A: 20}
				return pb.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Spacer{Height: unit.Dp(8)}.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				percentage := int(progress * 100)
				l := material.Caption(ui.theme, fmt.Sprintf("%d%% Complete", percentage))
				l.Color = CurrentTheme.TextLight
				l.Font.Typeface = "Montserrat"
				return l.Layout(gtx)
			}),
		)
	})
}

func drawCard(gtx layout.Context, content layout.Widget) layout.Dimensions {
	return widget.Border{Color: CurrentTheme.Border, CornerRadius: unit.Dp(8), Width: unit.Dp(1)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Stack{}.Layout(gtx,
			layout.Expanded(func(gtx layout.Context) layout.Dimensions {
				rr := gtx.Dp(unit.Dp(8))
				paint.FillShape(gtx.Ops, CurrentTheme.Surface, clip.RRect{
					Rect: image.Rectangle{Max: gtx.Constraints.Min},
					NE: rr, NW: rr, SE: rr, SW: rr,
				}.Op(gtx.Ops))
				return layout.Dimensions{Size: gtx.Constraints.Min}
			}),
			layout.Stacked(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(16), Bottom: unit.Dp(16), Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, content)
			}),
		)
	})
}

func drawPrimaryActionBtn(gtx layout.Context, btn *widget.Clickable, label string, onClick func()) layout.Dimensions {
	appState.mu.Lock()
	isProcessing := appState.isProcessing
	appState.mu.Unlock()

	if btn.Clicked(gtx) && !isProcessing {
		onClick()
	}

	bgColor := CurrentTheme.Primary
	if isProcessing {
		bgColor = CurrentTheme.TextLight
	}

	return btn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Stack{}.Layout(gtx,
			layout.Expanded(func(gtx layout.Context) layout.Dimensions {
				rr := gtx.Dp(unit.Dp(4))
				paint.FillShape(gtx.Ops, bgColor, clip.RRect{
					Rect: image.Rectangle{Max: gtx.Constraints.Min},
					NE: rr, NW: rr, SE: rr, SW: rr,
				}.Op(gtx.Ops))
				return layout.Dimensions{Size: gtx.Constraints.Min}
			}),
			layout.Stacked(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(12), Bottom: unit.Dp(12), Left: unit.Dp(32), Right: unit.Dp(32)}.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						txt := label
						if isProcessing {
							txt = "Processing..."
						}
						l := material.Body1(ui.theme, txt)
						l.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
						l.Font.Weight = font.Bold
						l.Font.Typeface = "Montserrat"
						l.Alignment = text.Middle
						return l.Layout(gtx)
					},
				)
			}),
		)
	})
}

// Backend Functions (Copy/Pasted and minimally adjusted for new UI state)
func addLog(msg string) {
	appState.mu.Lock()
	appState.logs = append(appState.logs, fmt.Sprintf("%s %s", time.Now().Format("15:04:05"), msg))
	appState.mu.Unlock()
	if window != nil {
		window.Invalidate()
	}
}

func updateStatus(status string) {
	appState.mu.Lock()
	appState.status = status
	appState.mu.Unlock()
	if window != nil { window.Invalidate() }
}

func updateProgress(progress float32) {
	appState.mu.Lock()
	appState.progress = progress
	appState.mu.Unlock()
	if window != nil { window.Invalidate() }
}

func updateSpeed(bytesProcessed int64) {
	appState.mu.Lock()
	elapsed := time.Since(appState.startTime).Seconds()
	if elapsed > 0 {
		speed := float64(bytesProcessed) / elapsed
		appState.speed = formatSpeed(speed)
	}
	appState.mu.Unlock()
	if window != nil { window.Invalidate() }
}

func performUpload(filePath string) {
	appState.mu.Lock()
	if appState.isProcessing {
		appState.mu.Unlock()
		return
	}
	appState.isProcessing = true
	appState.progress = 0
	appState.resultCID = ""
	appState.logs = make([]string, 0)
	appState.startTime = time.Now()
	encrypt := appState.encryptFile
	appState.mu.Unlock()
	
	window.Invalidate()

	defer func() {
		appState.mu.Lock()
		appState.isProcessing = false
		appState.mu.Unlock()
		window.Invalidate()
	}()

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		addLog("ERROR: " + err.Error())
		return
	}

	addLog(fmt.Sprintf("FILE: %s (%s)", filepath.Base(filePath), formatSize(fileInfo.Size())))
	addLog(fmt.Sprintf("ENCRYPTION: %v", encrypt))

	updateStatus("Reading file...")
	plaintext, err := os.ReadFile(filePath)
	if err != nil {
		addLog("ERROR reading file: " + err.Error())
		return
	}
	updateProgress(0.1)
	addLog("SUCCESS: File read")

	metadata := finalride.Metadata{
		Filename:  filepath.Base(filePath),
		Encrypted: encrypt,
	}

	var dataToUpload []byte

	if encrypt {
		key, err := finalride.GenerateKey()
		if err != nil {
			addLog("ERROR generating key: " + err.Error())
			return
		}
		updateStatus("Encrypting...")
		dataToUpload, err = finalride.EncryptData(plaintext, key)
		if err != nil {
			addLog("ERROR Encryption failed: " + err.Error())
			return
		}
		metadata.Key = base64.StdEncoding.EncodeToString(key)
		addLog("SUCCESS: Encryption complete")
	} else {
		dataToUpload = plaintext
	}
	updateProgress(0.3)

	chunkSizeBytes := config.ChunkSizeMB * 1024 * 1024

	if len(dataToUpload) > chunkSizeBytes {
		updateStatus("Chunking...")
		chunks, hashes := finalride.SplitIntoChunks(dataToUpload, chunkSizeBytes)
		addLog(fmt.Sprintf("SUCCESS: Split into %d chunks", len(chunks)))
		updateProgress(0.4)

		updateStatus("Uploading chunks...")
		chunkIDs := make(map[string]string)
		totalChunks := len(chunks)
		uploaded := 0

		for k, chunk := range chunks {
			ref, err := finalride.UploadToSwarm(chunk, config.SwarmAPI)
			if err != nil {
				addLog(fmt.Sprintf("ERROR upload chunk %s: %v", k, err))
				return
			}
			chunkIDs[k] = ref
			uploaded++
			updateProgress(0.4 + 0.5*float32(uploaded)/float32(totalChunks))
			updateSpeed(int64(uploaded * chunkSizeBytes))
		}
		metadata.Chunked = true
		metadata.ChunkIDs = chunkIDs
		metadata.ChunkHashes = hashes
		addLog("SUCCESS: All chunks uploaded")
	} else {
		updateStatus("Uploading...")
		fileID, err := finalride.UploadToSwarm(dataToUpload, config.SwarmAPI)
		if err != nil {
			addLog("ERROR Upload failed: " + err.Error())
			return
		}
		hash := sha256.Sum256(dataToUpload)
		metadata.Chunked = false
		metadata.FileID = fileID
		metadata.FileHash = fmt.Sprintf("%x", hash)
		addLog("SUCCESS: File uploaded")
	}
	updateProgress(0.9)

	updateStatus("Uploading metadata...")
	metadataJSON, _ := json.Marshal(metadata)
	metadataCID, err := finalride.UploadToSwarm(metadataJSON, config.SwarmAPI)
	if err != nil {
		addLog("ERROR upload metadata: " + err.Error())
		return
	}

	updateProgress(1.0)
	updateStatus("Complete!")
	addLog("SUCCESS: Upload complete!")
	addLog(fmt.Sprintf("CID: %s", metadataCID))

	appState.mu.Lock()
	appState.resultCID = metadataCID
	appState.mu.Unlock()
	window.Invalidate()
}

func performDownload(cid string) {
	appState.mu.Lock()
	if appState.isProcessing {
		appState.mu.Unlock()
		return
	}
	appState.isProcessing = true
	appState.progress = 0
	appState.logs = make([]string, 0)
	appState.startTime = time.Now()
	appState.mu.Unlock()
	
	window.Invalidate()

	defer func() {
		appState.mu.Lock()
		appState.isProcessing = false
		appState.mu.Unlock()
		window.Invalidate()
	}()

	addLog(fmt.Sprintf("Starting Download CID: %s", cid))

	updateStatus("Downloading metadata...")
	metadataJSON, err := finalride.DownloadFromSwarm(cid, config.SwarmAPI)
	if err != nil {
		addLog("ERROR metadata download: " + err.Error())
		return
	}
	updateProgress(0.1)

	var metadata finalride.Metadata
	if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
		addLog("ERROR parse metadata: " + err.Error())
		return
	}

	addLog(fmt.Sprintf("Info: %s (Encrypted: %v)", metadata.Filename, metadata.Encrypted))

	var downloadedData []byte

	if metadata.Chunked {
		updateStatus("Downloading chunks...")
		addLog(fmt.Sprintf("Downloading %d chunks...", len(metadata.ChunkIDs)))

		chunks := make(map[string][]byte)
		totalChunks := len(metadata.ChunkIDs)
		downloaded := 0

		for k, ref := range metadata.ChunkIDs {
			chunkData, err := finalride.DownloadFromSwarm(ref, config.SwarmAPI)
			if err != nil {
				addLog(fmt.Sprintf("ERROR download chunk %s: %v", k, err))
				return
			}
			hash := sha256.Sum256(chunkData)
			if metadata.ChunkHashes[k] != fmt.Sprintf("%x", hash) {
				addLog(fmt.Sprintf("ERROR Integrity failed chunk %s", k))
				return
			}
			chunks[k] = chunkData
			downloaded++
			updateProgress(0.1 + 0.6*float32(downloaded)/float32(totalChunks))
			updateSpeed(int64(downloaded * config.ChunkSizeMB * 1024 * 1024))
		}
		updateStatus("Reassembling...")
		downloadedData = finalride.ReassembleChunks(chunks)
	} else {
		updateStatus("Downloading file...")
		downloadedData, err = finalride.DownloadFromSwarm(metadata.FileID, config.SwarmAPI)
		if err != nil {
			addLog("ERROR Download failed: " + err.Error())
			return
		}
		hash := sha256.Sum256(downloadedData)
		if metadata.FileHash != fmt.Sprintf("%x", hash) {
			addLog("ERROR File integrity failed")
			return
		}
	}
	updateProgress(0.8)

	var finalData []byte
	if metadata.Encrypted {
		updateStatus("Decrypting...")
		key, err := base64.StdEncoding.DecodeString(metadata.Key)
		if err != nil {
			addLog("ERROR decode key: " + err.Error())
			return
		}
		finalData, err = finalride.DecryptData(downloadedData, key)
		if err != nil {
			addLog("ERROR Decryption: " + err.Error())
			return
		}
		addLog("SUCCESS: Decrypted")
	} else {
		finalData = downloadedData
	}
	updateProgress(0.9)

	updateStatus("Saving file...")
	savePath := metadata.Filename
	if config.DownloadDir != "" {
		savePath = filepath.Join(config.DownloadDir, metadata.Filename)
	}
	if err := os.WriteFile(savePath, finalData, 0644); err != nil {
		addLog("ERROR Save file: " + err.Error())
		return
	}

	updateProgress(1.0)
	updateStatus("Complete!")
	addLog(fmt.Sprintf("SUCCESS: Saved %s (%s)", savePath, formatSize(int64(len(finalData)))))
}

func formatSpeed(bytesPerSec float64) string {
	if bytesPerSec >= 1024*1024*1024 {
		return fmt.Sprintf("%.2f GB/s", bytesPerSec/(1024*1024*1024))
	} else if bytesPerSec >= 1024*1024 {
		return fmt.Sprintf("%.2f MB/s", bytesPerSec/(1024*1024))
	} else if bytesPerSec >= 1024 {
		return fmt.Sprintf("%.2f KB/s", bytesPerSec/1024)
	}
	return fmt.Sprintf("%.2f B/s", bytesPerSec)
}

func formatSize(bytes int64) string {
	if bytes >= 1024*1024*1024 {
		return fmt.Sprintf("%.2f GB", float64(bytes)/(1024*1024*1024))
	} else if bytes >= 1024*1024 {
		return fmt.Sprintf("%.2f MB", float64(bytes)/(1024*1024))
	} else if bytes >= 1024 {
		return fmt.Sprintf("%.2f KB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%d B", bytes)
}
