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
		Name: "aws_logo",
		Frames: []string{
			`
           ,*.
          /   '.
         /      '.
        /    /'.   '.
       /    /   '.   '.
      /    /      '.   '.
     /    /         '.   '.
         /            '.
        /               '.
       '-------------------'

      a m a z o n  w e b
      s e r v i c e s
`,
			`
           ,*.
          / * '.
         / * * '.
        / *  /'. * '.
       / *  /   '.  *'.
      / *  /      '. * '.
     / *  /       * '.   '.
      *  /          * '.
       */             * '.
       '-------------------'
    *  a m a z o n  w e b  *
    *  s e r v i c e s     *
`,
			`
           ,*.
          /   '.
         /      '.
        /    /'.   '.
       /    /   '.   '.
      /    /      '.   '.
     /    /         '.   '.
         /            '.
        /               '.
       '-------------------'
    >> a m a z o n  w e b <<
    >> s e r v i c e s   <<
`,
			`
           ,*.
          / * '.
         / * * '.
        / *  /'. * '.
       / *  /   '.  *'.
      / *  /      '. * '.
     / *  /       * '.   '.
      *  /          * '.
       */             * '.
       '-------------------'

      a m a z o n  w e b
      s e r v i c e s
`,
		},
	},
	{
		Name: "cloud_deploy",
		Frames: []string{
			`
              .-~~~-.
      .- ~ ~-(       )- ~ ~ -.
     /                         \
    |      A W S  C L O U D     |
     \                         /
      ~- . _____________ . -~

         |               |
         |   [deploying] |
         |   [ ....    ] |
         |_______________|
`,
			`
              .-~~~-.
      .- ~ ~-(       )- ~ ~ -.
     /                         \
    |      A W S  C L O U D     |
     \                         /
      ~- . _____________ . -~
             |       |
         ____|_______|____
         |               |
         |   [deploying] |
         |   [ =====.. ] |
         |_______________|
`,
			`
              .-~~~-.
      .- ~ ~-(       )- ~ ~ -.
     /                         \
    |      A W S  C L O U D     |
     \           *             /
      ~- . _____________ . -~
           * |       | *
         ____|_______|____
         |               |
         |   [deploying] |
         |   [ ======== ]|
         |_______________|
`,
			`
              .-~~~-.
      .- ~ ~-(       )- ~ ~ -.
     /          * *             \
    |      A W S  C L O U D     |
     \        * * * *          /
      ~- . _____________ . -~
          ** |       | **
         ____|_______|____
         |               |
         |   [ success ] |
         |   [==========]|
         |_______________|
`,
		},
	},
	{
		Name: "lambda_invoke",
		Frames: []string{
			`
        _                 _         _
       | |               | |       | |
       | | __ _ _ __ ___ | |__   __| | __ _
       | |/ _' | '_ ' _ \| '_ \ / _' |/ _' |
       | | (_| | | | | | | |_) | (_| | (_| |
       |_|\__,_|_| |_| |_|_.__/ \__,_|\__,_|

               invoking .
`,
			`
        _                 _         _
       | |               | |       | |
       | | __ _ _ __ ___ | |__   __| | __ _
       | |/ _' | '_ ' _ \| '_ \ / _' |/ _' |
       | | (_| | | | | | | |_) | (_| | (_| |
       |_|\__,_|_| |_| |_|_.__/ \__,_|\__,_|

               invoking . .
`,
			`
        _                 _         _
       | |               | |       | |
       | | __ _ _ __ ___ | |__   __| | __ _
       | |/ _' | '_ ' _ \| '_ \ / _' |/ _' |
       | | (_| | | | | | | |_) | (_| | (_| |
       |_|\__,_|_| |_| |_|_.__/ \__,_|\__,_|

               invoking . . .
`,
			`
        _                 _         _
       | |               | |       | |
       | | __ _ _ __ ___ | |__   __| | __ _
       | |/ _' | '_ ' _ \| '_ \ / _' |/ _' |
       | | (_| | | | | | | |_) | (_| | (_| |
       |_|\__,_|_| |_| |_|_.__/ \__,_|\__,_|

            >> 200 OK <<
`,
		},
	},
	{
		Name: "s3_bucket",
		Frames: []string{
			`
            _.-----._
          .'          '.
         /  Amazon  S3  \
        |    ________    |
        |   |        |   |
        |   | []     |   |
        |   | []     |   |
        |   | []     |   |
        |   |________|   |
         \              /
          '._________.'

        loading buckets .
`,
			`
            _.-----._
          .'          '.
         /  Amazon  S3  \
        |    ________    |
        |   |        |   |
        |   | [] []  |   |
        |   | [] []  |   |
        |   | [] []  |   |
        |   |________|   |
         \              /
          '._________.'

        loading buckets . .
`,
			`
            _.-----._
          .'          '.
         /  Amazon  S3  \
        |    ________    |
        |   |  * * * |   |
        |   | [][][] |   |
        |   | [][][] |   |
        |   | [][][] |   |
        |   |________|   |
         \              /
          '._________.'

        loading buckets . . .
`,
			`
            _.-----._
          .'    **    '.
         /  Amazon  S3  \
        |    ________    |
        |   | ** * **|   |
        |   | [][][] |   |
        |   | [][][] |   |
        |   | [][][] |   |
        |   |________|   |
         \              /
          '._________.'

        buckets ready!
`,
		},
	},
	{
		Name: "ec2_instance",
		Frames: []string{
			`
      +------------------------------+
      |  _____  ___  ____            |
      | | ____||__ \|___ \           |
      | |  _|    ) | __) |          |
      | | |___  / / / __/           |
      | |_____|/___|_____|          |
      |                              |
      |  Instance: i-0a1b2c3d       |
      |  Status:  pending            |
      +------------------------------+
`,
			`
      +------------------------------+
      |  _____  ___  ____            |
      | | ____||__ \|___ \           |
      | |  _|    ) | __) |          |
      | | |___  / / / __/           |
      | |_____|/___|_____|          |
      |                              |
      |  Instance: i-0a1b2c3d       |
      |  Status:  initializing ..    |
      +------------------------------+
`,
			`
      +------------------------------+
      |  _____  ___  ____            |
      | | ____||__ \|___ \           |
      | |  _|    ) | __) |          |
      | | |___  / / / __/           |
      | |_____|/___|_____|          |
      |                              |
      |  Instance: i-0a1b2c3d       |
      |  Status:  * running *        |
      +------------------------------+
`,
			`
      +------------------------------+
      |  _____  ___  ____            |
      | | ____||__ \|___ \           |
      | |  _|    ) | __) |          |
      | | |___  / / / __/           |
      | |_____|/___|_____|          |
      |                              |
      |  Instance: i-0a1b2c3d       |
      |  Status: ** running **       |
      +------------------------------+
`,
		},
	},
	{
		Name: "cloudformation_stack",
		Frames: []string{
			`
     C L O U D F O R M A T I O N

     +----------+  +----------+
     | Resource |  | Resource |
     |    #1    |  |    #2    |
     +----++----+  +----++----+
          ||            ||
          ++----+-------++
                |
          +-----+------+
          |  Resource   |
          |     #3      |
          +-------------+

          building stack .
`,
			`
     C L O U D F O R M A T I O N

     +----------+  +----------+
     | Resource |  | Resource |
     |  * #1 *  |  |    #2    |
     +----++----+  +----++----+
          ||            ||
          ++----+-------++
                |
          +-----+------+
          |  Resource   |
          |     #3      |
          +-------------+

          building stack . .
`,
			`
     C L O U D F O R M A T I O N

     +----------+  +----------+
     | Resource |  | Resource |
     |  * #1 *  |  |  * #2 *  |
     +----++----+  +----++----+
          ||            ||
          ++----+-------++
                |
          +-----+------+
          |  Resource   |
          |     #3      |
          +-------------+

          building stack . . .
`,
			`
     C L O U D F O R M A T I O N

     +----------+  +----------+
     | Resource |  | Resource |
     |  * #1 *  |  |  * #2 *  |
     +----++----+  +----++----+
          ||            ||
          ++----+-------++
                |
          +-----+------+
          |  Resource   |
          |   * #3 *    |
          +-------------+

        CREATE_COMPLETE!
`,
		},
	},
	{
		Name: "dynamodb_table",
		Frames: []string{
			`
      +-----------------------------------+
      |         D y n a m o D B           |
      +-----------------------------------+
      | PK       | SK       | Data        |
      |----------+----------+-------------|
      |          |          |             |
      |          |          |             |
      |          |          |             |
      |          |          |             |
      +-----------------------------------+
               scanning table .
`,
			`
      +-----------------------------------+
      |         D y n a m o D B           |
      +-----------------------------------+
      | PK       | SK       | Data        |
      |----------+----------+-------------|
      | user#001 |          |             |
      |          |          |             |
      |          |          |             |
      |          |          |             |
      +-----------------------------------+
               scanning table . .
`,
			`
      +-----------------------------------+
      |         D y n a m o D B           |
      +-----------------------------------+
      | PK       | SK       | Data        |
      |----------+----------+-------------|
      | user#001 | meta     | { ... }     |
      | user#002 | meta     |             |
      |          |          |             |
      |          |          |             |
      +-----------------------------------+
               scanning table . . .
`,
			`
      +-----------------------------------+
      |         D y n a m o D B           |
      +-----------------------------------+
      | PK       | SK       | Data        |
      |----------+----------+-------------|
      | user#001 | meta     | { ... }     |
      | user#002 | meta     | { ... }     |
      | user#003 | order#01 | { ... }     |
      | user#004 | meta     | { ... }     |
      +-----------------------------------+
              scan complete!
`,
		},
	},
	{
		Name: "vpc_network",
		Frames: []string{
			`
      +========================================+
      |              A W S  V P C              |
      |  +----------+        +----------+      |
      |  | Subnet A |        | Subnet B |      |
      |  |          |        |          |      |
      |  |  [EC2]   |        |  [EC2]   |      |
      |  |          |        |          |      |
      |  +----------+        +----------+      |
      |                                        |
      |        Internet Gateway: ...           |
      +========================================+
`,
			`
      +========================================+
      |              A W S  V P C              |
      |  +----------+        +----------+      |
      |  | Subnet A | -----> | Subnet B |      |
      |  |          |        |          |      |
      |  |  [EC2]   |        |  [EC2]   |      |
      |  |          |        |          |      |
      |  +----------+        +----------+      |
      |                                        |
      |        Internet Gateway: ...           |
      +========================================+
`,
			`
      +========================================+
      |              A W S  V P C              |
      |  +----------+        +----------+      |
      |  | Subnet A | <====> | Subnet B |      |
      |  |          |        |          |      |
      |  |  [EC2]   |  ***   |  [EC2]   |      |
      |  |          |        |          |      |
      |  +----------+        +----------+      |
      |        |                   |           |
      |        Internet Gateway: active        |
      +========================================+
`,
			`
      +========================================+
      |              A W S  V P C              |
      |  +----------+ *    * +----------+      |
      |  | Subnet A | <====> | Subnet B |      |
      |  |   *  *   |        |   *  *   |      |
      |  |  [EC2]   | <***>  |  [EC2]   |      |
      |  |          |        |          |      |
      |  +----------+        +----------+      |
      |        |          |        |           |
      |     ** Internet Gateway: active **     |
      +========================================+
`,
		},
	},
	{
		Name: "iam_shield",
		Frames: []string{
			`
              /\
             /  \
            / I  \
           / A    \
          / M      \
         /  ________\
        / /          \ \
       / /   SECURE   \ \
      / /               \ \
     / /                 \ \
    / /___________________\ \
    \/                     \/

       authenticating .
`,
			`
              /\
             /**\
            /* I *\
           /* A  * \
          /* M    * \
         /*  ________\
        / /    *     * \
       / /   SECURE   \ \
      / /       *       \ \
     / /     *       *   \ \
    / /___________________\ \
    \/                     \/

       authenticating . .
`,
			`
              /\
             /  \
            / I  \
           / A    \
          / M      \
         /  ________\
        / /          \ \
       / /   SECURE   \ \
      / /               \ \
     / /                 \ \
    / /___________________\ \
    \/                     \/

       authenticating . . .
`,
			`
              /\
             /**\
            /* I *\
           /* A  * \
          /* M    * \
         /*  ________\
        /*/ ** ** ** * \
       / /   SECURE   \ \
      / / ** ** ** ** ** \ \
     / /  **  ** **  **   \ \
    / /___________________\ \
    \/                     \/

       ** verified! **
`,
		},
	},
	{
		Name: "sqs_queue",
		Frames: []string{
			`
       +------+   +------+   +------+
       | msg  |   | msg  |   | msg  |
       |  #1  |   |  #2  |   |  #3  |
       +--+---+   +--+---+   +--+---+
          |           |           |
          v           v           v
      +----------------------------------+
      |     A M A Z O N   S Q S          |
      |                                  |
      |  >>>  [          queue ] >>>     |
      |                                  |
      +----------------------------------+

           processing .
`,
			`
       +------+   +------+   +------+
       | msg  |-->| msg  |-->| msg  |
       |  #1  |   |  #2  |   |  #3  |
       +--+---+   +--+---+   +--+---+
          |           |           |
          v           v           v
      +----------------------------------+
      |     A M A Z O N   S Q S          |
      |                                  |
      |  >>>  [ ==       queue ] >>>     |
      |                                  |
      +----------------------------------+

           processing . .
`,
			`
       +------+   +------+   +------+
       | msg  |-->| msg  |-->| msg  |
       |  #1  |   |  #2  |   |  #3  |
       +--+---+   +--+---+   +--+---+
          |           |           |
          v           v           v
      +----------------------------------+
      |     A M A Z O N   S Q S          |
      |                                  |
      |  >>>  [ =======  queue ] >>>     |
      |                                  |
      +----------------------------------+

           processing . . .
`,
			`
       +------+   +------+   +------+
       | msg  |-->| msg  |-->| msg  |
       |  #1  |   |  #2  |   |  #3  |
       +--+---+   +--+---+   +--+---+
          |           |           |
          v           v           v
      +----------------------------------+
      |     A M A Z O N   S Q S          |
      |                                  |
      |  >>>  [============== ] >>>      |
      |                                  |
      +----------------------------------+

           all messages delivered!
`,
		},
	},
	{
		Name: "cloudwatch_monitor",
		Frames: []string{
			`
     C L O U D W A T C H

     +------------------------------------+
     |                                    |
     |                             _      |
     |                            | |     |
     |                    _       | |     |
     |          _        | |      | |     |
     |    _    | |       | |      | |     |
     |   | |   | |  _    | |  _   | |     |
     |   | |   | | | |   | | | |  | |     |
     +------------------------------------+

           monitoring .
`,
			`
     C L O U D W A T C H

     +------------------------------------+
     |                          _         |
     |                         | |  _     |
     |                    _    | | | |    |
     |          _        | |   | | | |    |
     |    _    | |       | |   | | | |    |
     |   | |   | |  _    | |   | | | |    |
     |   | |   | | | |   | |   | | | |    |
     |   | |   | | | |   | |   | | | |    |
     +------------------------------------+

           monitoring . .
`,
			`
     C L O U D W A T C H

     +------------------------------------+
     |    _                               |
     |   | |  _                    _      |
     |   | | | |  _               | |     |
     |   | | | | | |     _        | |     |
     |   | | | | | |    | |  _    | |     |
     |   | | | | | |    | | | |   | |     |
     |   | | | | | |    | | | |   | |     |
     |   | | | | | |    | | | |   | |     |
     +------------------------------------+

           monitoring . . .
`,
			`
     C L O U D W A T C H

     +------------------------------------+
     |              _                     |
     |    _        | | _           _      |
     |   | |  _    | || |  _      | |     |
     |   | | | |   | || | | |  _  | |     |
     |   | | | |   | || | | | | | | |     |
     |   | | | |   | || | | | | | | |     |
     |   | | | |   | || | | | | | | |     |
     |   | | | |   | || | | | | | | |     |
     +------------------------------------+

         ** all healthy **
`,
		},
	},
	{
		Name: "api_gateway",
		Frames: []string{
			`
          A P I   G A T E W A Y

      client                    service
        |                          |
        |  --- GET /resource -->   |
        |                          |
        |                          |
        |                          |
        |                          |
`,
			`
          A P I   G A T E W A Y

      client        [GW]       service
        |            |            |
        |  ------->  |            |
        |            |  ------->  |
        |            |            |
        |            |            |
        |            |            |
`,
			`
          A P I   G A T E W A Y

      client        [GW]       service
        |            |            |
        |  ------->  |            |
        |            |  ------->  |
        |            |            |
        |            |  <-------  |
        |  <-------  |            |
`,
			`
          A P I   G A T E W A Y

      client        [GW]       service
        |            |            |
        |  ------->  |            |
        |            |  ------->  |
        |            |    200 OK  |
        |            |  <-------  |
        |   200 OK   |            |
        |  <-------  |            |

               ** complete **
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
				AddItem(textView, 60, 0, false).
				AddItem(nil, 0, 1, false),
			20, 0, false).
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
