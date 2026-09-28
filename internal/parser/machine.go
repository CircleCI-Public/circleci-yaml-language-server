package parser

const machineTrueAdvice = "CircleCI advises against using \"machine: true\", as support for this feature is not guaranteed to continue in the future.\n\n"

func MachineTrueMessage(img string) string {
	return machineTrueAdvice + "You can replace it with the following explicit declaration, which uses the same image.\n" +
		"machine:\n" +
		"  image: " + img
}

// MachineTrueWindowsMessage names no image: the one `machine: true` gets on a
// Windows resource class isn't known here, and the Linux default would be
// wrong.
const MachineTrueWindowsMessage = machineTrueAdvice + `Declare one of the Windows machine images instead. See https://circleci.com/docs/reference/configuration-reference/#available-windows-machine-images`
