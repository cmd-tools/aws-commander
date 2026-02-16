package ui

import (
	"math/rand"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const animationFrameDelay = 200 * time.Millisecond
const animationTotalDuration = 2 * time.Second

// Animation holds multiple frames of ASCII art to cycle through.
type Animation struct {
	Name   string
	Frames []string
}

// animations is the pool of available ASCII animations. One is picked at random each time.
var animations = []Animation{
	{
		Name: "rocket",
		Frames: []string{
			`
        *
       /|\
      / | \
     /  |  \
    /___|___\
       |||
       |||
      /   \
     / AWS \
    /_______\
`,
			`
        *
       /|\
      / | \
     /  |  \
    /___|___\
       |||
       |||
      /   \
     / AWS \
    /_______\
      ^ ^ ^
`,
			`
        *
       /|\
      / | \
     /  |  \
    /___|___\
       |||
       |||
      /   \
     / AWS \
    /_______\
     ^^ ^ ^^
      *   *
`,
			`
        *
       /|\
      / | \
     /  |  \
    /___|___\
       |||
       |||
      /   \
     / AWS \
    /_______\
    ^^ ^ ^ ^^
     * * * *
      *   *
`,
		},
	},
	{
		Name: "cloud",
		Frames: []string{
			`
          ___
      ___/   \___
    /             \
   |    A W S      |
    \  COMMANDER  /
      \___   ___/
          \_/
`,
			`
         .___. 
      ___/   \___
    /             \
   |    A W S      |
    \  COMMANDER  /
      \___   ___/
          \_/
           .
`,
			`
        ..___.. 
      ___/   \___
    /             \
   |    A W S      |
    \  COMMANDER  /
      \___   ___/
          \_/
          ...
`,
			`
       ...___... 
      ___/   \___
    /             \
   |    A W S      |
    \  COMMANDER  /
      \___   ___/
          \_/
         .....
`,
		},
	},
	{
		Name: "loading_bar",
		Frames: []string{
			`
   +-----------------------+
   |                       |
   |   [#                ] |
   |                       |
   |     Loading . . .     |
   +-----------------------+
`,
			`
   +-----------------------+
   |                       |
   |   [####             ] |
   |                       |
   |     Loading . . .     |
   +-----------------------+
`,
			`
   +-----------------------+
   |                       |
   |   [########         ] |
   |                       |
   |     Loading . . .     |
   +-----------------------+
`,
			`
   +-----------------------+
   |                       |
   |   [############     ] |
   |                       |
   |     Loading . . .     |
   +-----------------------+
`,
			`
   +-----------------------+
   |                       |
   |   [################]] |
   |                       |
   |     Loading . . .     |
   +-----------------------+
`,
			`
   +-----------------------+
   |                       |
   |   [###################|
   |                       |
   |        Ready!         |
   +-----------------------+
`,
		},
	},
	{
		Name: "satellite",
		Frames: []string{
			`
           .
          .:.
         .:::.
           |
           |
     ______|______
    |  --=====--  |
    |_____________|
       |      |
      /        \
`,
			`
           *
          *:*
         *:::*
           |
           |
     ______|______
    |  --=====--  |
    |_____________|
       |      |
      /        \
`,
			`
           .
          .:. 
         .:::.
           |
           |
     ______|______
    |  --=====--  |
    |_____________|
       |      |
      /   ..   \
`,
			`
           *
          *:* 
         *:::*
           |
           |
     ______|______
    |  --=====--  |
    |_____________|
       |      |
      /  ....  \
`,
		},
	},
	{
		Name: "spinner",
		Frames: []string{
			`
      ___________
     |           |
     |    |      |
     |    |      |
     |           |
     |___________|
    AWS  COMMANDER
`,
			`
      ___________
     |           |
     |     /     |
     |    /      |
     |           |
     |___________|
    AWS  COMMANDER
`,
			`
      ___________
     |           |
     |    ---    |
     |           |
     |           |
     |___________|
    AWS  COMMANDER
`,
			`
      ___________
     |           |
     |      \    |
     |       \   |
     |           |
     |___________|
    AWS  COMMANDER
`,
			`
      ___________
     |           |
     |    |      |
     |    |      |
     |           |
     |___________|
    AWS  COMMANDER
`,
		},
	},
	{
		Name: "radar",
		Frames: []string{
			`
        .---.
       /     \
      |   |   |
       \     /
        '---'
     scanning...
`,
			`
        .---.
       /  /  \
      |  /    |
       \/    /
        '---'
     scanning...
`,
			`
        .---.
       / --- \
      |  ---  |
       \ --- /
        '---'
     scanning...
`,
			`
        .---.
       /     \
      |    \  |
       \    \/
        '---'
     scanning...
`,
		},
	},
	{
		Name: "terminal",
		Frames: []string{
			`
    +------------------+
    | $ aws _          |
    |                  |
    |                  |
    +------------------+
`,
			`
    +------------------+
    | $ aws command_   |
    |                  |
    |                  |
    +------------------+
`,
			`
    +------------------+
    | $ aws commander  |
    | > connecting...  |
    |                  |
    +------------------+
`,
			`
    +------------------+
    | $ aws commander  |
    | > connecting...  |
    | > ready!         |
    +------------------+
`,
		},
	},
	{
		Name: "wave",
		Frames: []string{
			`
    ~                     ~
     ~   AWS COMMANDER   ~
    ~                     ~
`,
			`
     ~                   ~
    ~    AWS COMMANDER    ~
     ~                   ~
`,
			`
    ~                     ~
     ~   AWS COMMANDER   ~
    ~                     ~
`,
			`
      ~                 ~
     ~  AWS COMMANDER  ~
      ~                 ~
`,
		},
	},
	{
		Name: "pulse",
		Frames: []string{
			`

         < * >
       A W S
     COMMANDER

`,
			`

        <  *  >
        A W S
      COMMANDER

`,
			`

       <   *   >
       A  W  S
     C O M M A N D E R

`,
			`

        <  *  >
        A W S
      COMMANDER

`,
		},
	},
	{
		Name: "server_rack",
		Frames: []string{
			`
    +-----------+
    | [=] o   o |
    | [=] o   o |
    | [=] o   o |
    | [=] o   o |
    +-----------+
      | |   | |
`,
			`
    +-----------+
    | [=] *   o |
    | [=] o   * |
    | [=] *   o |
    | [=] o   * |
    +-----------+
      | |   | |
`,
			`
    +-----------+
    | [=] o   * |
    | [=] *   o |
    | [=] o   * |
    | [=] *   o |
    +-----------+
      | |   | |
`,
			`
    +-----------+
    | [=] *   * |
    | [=] *   * |
    | [=] *   * |
    | [=] *   * |
    +-----------+
      | |   | |
`,
		},
	},
	{
		Name: "globe",
		Frames: []string{
			`
        .-""-.
       /      \
      |  ----  |
      |  ----  |
       \      /
        '-__-'
`,
			`
        .-""-.
       / --   \
      | --     |
      |   --   |
       \    --/
        '-__-'
`,
			`
        .-""-.
       /   -- \
      |    -- |
      | --    |
       \--   /
        '-__-'
`,
			`
        .-""-.
       /      \
      |  ----  |
      |  ----  |
       \      /
        '-__-'
`,
		},
	},
	{
		Name: "lightning",
		Frames: []string{
			`
       \
        \
         \
          \
           \
            \
`,
			`
       \  /
        \/
        /\
       /  \
      /    \
     /      \
`,
			`
     * \  / *
      * \/ *
       */\*
      */ *\*
     */    \*
    */   *  \*
`,
			`
    ** \  / **
     ** \/ **
      **/\**
     **/ \**
    **/    \**
   **/   *  \**
`,
		},
	},
}

// RandomAnimation returns a randomly selected animation from the pool.
func RandomAnimation() Animation {
	return animations[rand.Intn(len(animations))]
}

// createAnimationView builds the centered text view and container used by all animation functions.
func createAnimationView() (*tview.TextView, *tview.Flex) {
	textView := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(false)
	textView.SetBackgroundColor(tcell.ColorDefault)
	textView.SetBorderPadding(1, 1, 2, 2)

	centered := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(
			tview.NewFlex().
				AddItem(nil, 0, 1, false).
				AddItem(textView, 40, 0, false).
				AddItem(nil, 0, 1, false),
			14, 0, false).
		AddItem(nil, 0, 1, false)
	centered.SetBackgroundColor(tcell.ColorDefault)

	return textView, centered
}

// ShowAnimationOverlay displays a random ASCII animation as a centered overlay on top of
// the provided background. The animation plays for animationTotalDuration then auto-dismisses
// by calling onDone. Returns a tview.Pages primitive to be used as the root/body.
func ShowAnimationOverlay(app *tview.Application, background tview.Primitive, onDone func()) *tview.Pages {
	anim := RandomAnimation()
	textView, centered := createAnimationView()

	pages := tview.NewPages().
		AddPage("background", background, true, true).
		AddPage("animation", centered, true, true)

	// Allow ESC to dismiss early
	dismissed := false
	centered.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc || event.Key() == tcell.KeyEnter {
			if !dismissed {
				dismissed = true
				onDone()
			}
			return nil
		}
		return event
	})

	// Start animation loop
	go func() {
		frameCount := len(anim.Frames)
		totalFrames := int(animationTotalDuration / animationFrameDelay)

		for i := 0; i < totalFrames; i++ {
			if dismissed {
				return
			}
			frame := anim.Frames[i%frameCount]
			app.QueueUpdateDraw(func() {
				textView.SetText(frame)
			})
			time.Sleep(animationFrameDelay)
		}

		if !dismissed {
			dismissed = true
			app.QueueUpdateDraw(func() {
				onDone()
			})
		}
	}()

	return pages
}

// ShowLoadingAnimation displays a random ASCII animation as a full-screen view that loops
// indefinitely until the done channel is closed. Once done is signalled, onDone is called
// via QueueUpdateDraw to swap the view. Returns the Flex primitive to use as root.
func ShowLoadingAnimation(app *tview.Application, done <-chan struct{}, onDone func()) *tview.Flex {
	anim := RandomAnimation()
	textView, centered := createAnimationView()

	go func() {
		frameCount := len(anim.Frames)
		i := 0
		for {
			select {
			case <-done:
				app.QueueUpdateDraw(func() {
					onDone()
				})
				return
			default:
				frame := anim.Frames[i%frameCount]
				app.QueueUpdateDraw(func() {
					textView.SetText(frame)
				})
				time.Sleep(animationFrameDelay)
				i++
			}
		}
	}()

	return centered
}
