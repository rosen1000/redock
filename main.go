package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/gdamore/tcell/v2"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

var defaultStyle = tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)
var invertedStyle = tcell.StyleDefault.Background(tcell.ColorLightGray).Foreground(tcell.ColorBlack)
var boldStyle = tcell.StyleDefault.Bold(true)
var boxStyle = tcell.StyleDefault.Foreground(tcell.ColorGreen)

var width, height int
var screen, screenErr = tcell.NewScreen()

func main() {
	// drawBox(Point{0, 0}, Point{width - 1, height - 11}, boxStyle)
	mainBox := Box{Point{0, 0}, Point{width - 1, height - 11}}
	mainBox.render()

	containers := listContainers()
	nameLen := 0
	for _, c := range containers {
		l := len(c.Image)
		if nameLen < l {
			nameLen = l
		}
	}
	mainBox.drawTextSimple(Point{0, 0}, boldStyle, "IMAGE")
	mainBox.drawTextSimple(Point{nameLen + 1, 0}, boldStyle, "PORTS")
	for i, container := range containers {
		name := container.Image
		var port []string
		for _, p := range container.Ports {
			if p.IP != "::" {
				port = append(port, fmt.Sprintf("%v -> %v %v", p.PrivatePort, p.PublicPort, p.Type))
			}
		}
		used(i)
		used(strings.TrimSpace(name))
		mainBox.drawTextSimple(Point{0, 1 + i}, defaultStyle, name)
		mainBox.drawTextSimple(Point{1 + nameLen, 1 + i}, defaultStyle, strings.Join(port, ", "))
	}

	selectLine := 0
	setStyle(Point{1, selectLine + 2}, Point{width - 2, selectLine + 2}, invertedStyle)

	infoBox := Box{Point{0, height - 10}, Point{width - 1, height - 1}}
	renderInfo(containers[0], infoBox)
	for {
		screen.Show()

		ev := screen.PollEvent()
		switch ev := ev.(type) {
		// case *tcell.EventMouse:
		// switch ev.Buttons() {
		// case tcell.Button1, tcell.Button2:
		// 	x, y := ev.Position()
		// 	screen.SetContent(x, y, ' ', nil, tcell.StyleDefault.Background(tcell.ColorWhite))
		// }
		case *tcell.EventResize:
			width, height = screen.Size()
			screen.Sync()
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyUp:
				if selectLine == 0 {
					continue
				}
				setStyle(Point{1, selectLine + 2}, Point{width - 2, selectLine + 2}, defaultStyle)
				selectLine--
				setStyle(Point{1, selectLine + 2}, Point{width - 2, selectLine + 2}, invertedStyle)
				renderInfo(containers[selectLine], infoBox)
			case tcell.KeyDown:
				if selectLine == len(containers)-1 {
					continue
				}
				setStyle(Point{1, selectLine + 2}, Point{width - 2, selectLine + 2}, defaultStyle)
				selectLine++
				setStyle(Point{1, selectLine + 2}, Point{width - 2, selectLine + 2}, invertedStyle)
				renderInfo(containers[selectLine], infoBox)
			case tcell.KeyEscape, tcell.KeyCtrlC:
				quit()
			}
		}
	}
}

func init() {
	catch(screenErr)
	catch(screen.Init())
	width, height = screen.Size()

	screen.SetStyle(defaultStyle)
	screen.Clear()
	screen.EnableMouse()
}

func quit() {
	r := recover()
	screen.Fini()
	if r != nil {
		panic(r)
	}
	os.Exit(0)
}

func renderInfo(cont types.Container, infoBox Box) {
	infoBox.render()
	infoBox.drawTextSimple(Point{0, 0}, defaultStyle, cont.Image)
}

func drawTextSimple(start Point, style tcell.Style, text string) {
	drawText(start, Point{start.x + len(text), start.y}, style, text)
}

func drawText(p1, p2 Point, style tcell.Style, text string) {
	row := p1.y
	col := p1.x
	for _, r := range []rune(text) {
		screen.SetContent(col, row, r, nil, style)
		col++
		if col >= p2.x {
			row++
			col = p1.x
		}
		if row > p2.y {
			break
		}
	}
}

func setStyle(p1, p2 Point, style tcell.Style) {
	p1, p2 = p1.compare(p2)

	for x := p1.x; x <= p2.x; x++ {
		for y := p1.y; y <= p2.y; y++ {
			r, c, _, _ := screen.GetContent(x, y)
			screen.SetContent(x, y, r, c, style)
		}
	}
}

func drawBox(p1, p2 Point, style tcell.Style) {
	p1, p2 = p1.compare(p2)
	x1, y1, x2, y2 := p1.x, p1.y, p2.x, p2.y

	for col := x1; col <= x2; col++ {
		for row := y1 + 1; row < y2; row++ {
			screen.SetContent(col, row, ' ', nil, style)
		}
	}

	// Draw borders
	for col := x1; col <= x2; col++ {
		screen.SetContent(col, y1, tcell.RuneHLine, nil, style)
		screen.SetContent(col, y2, tcell.RuneHLine, nil, style)
	}
	for row := y1 + 1; row < y2; row++ {
		screen.SetContent(x1, row, tcell.RuneVLine, nil, style)
		screen.SetContent(x2, row, tcell.RuneVLine, nil, style)
	}

	// Only draw corners if necessary
	if y1 != y2 && x1 != x2 {
		screen.SetContent(x1, y1, tcell.RuneULCorner, nil, style)
		screen.SetContent(x2, y1, tcell.RuneURCorner, nil, style)
		screen.SetContent(x1, y2, tcell.RuneLLCorner, nil, style)
		screen.SetContent(x2, y2, tcell.RuneLRCorner, nil, style)
	}
}

func listContainers() []types.Container {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}

	return containers
}

type Point struct {
	x int
	y int
}

func (p1 Point) compare(p2 Point) (Point, Point) {
	if p1.y > p2.y {
		p1.y, p2.y = p2.y, p1.y
	}
	if p1.x > p2.x {
		p1.x, p2.x = p2.x, p1.x
	}
	return p1, p2
}

func (p1 Point) add(p2 Point) Point {
	return Point{p1.x + p2.x, p1.y + p2.y}
}

type Box struct {
	start, end Point
}

var boxOff = Point{1, 1}

func (b Box) drawText(p1, p2 Point, style tcell.Style, text string) {
	drawText(b.start.add(p1).add(boxOff), b.end.add(p2).add(boxOff), style, text)
}

func (b Box) drawTextSimple(p Point, style tcell.Style, text string) {
	drawTextSimple(b.start.add(p).add(boxOff), style, text)
}

func (b Box) render() {
	drawBox(b.start, b.end, boxStyle)
}

func used(val any) {
	_ = val
}

func catch(err error) {
	if err != nil {
		panic(err)
	}
}
