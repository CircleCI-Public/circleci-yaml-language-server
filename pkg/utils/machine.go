package utils

func GetMachineTrueMessage(img string) string {
	return `CircleCI advises against using "machine: true", as support for this feature is not guaranteed to continue in the future.
		
	You can replace it with the following explicit declaration, which uses the same image.
	machine:
		image: ` + img
}
