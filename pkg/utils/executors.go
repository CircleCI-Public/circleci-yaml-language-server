package utils

import (
	"slices"
	"strings"
)

var CurrentLinuxImage = "ubuntu-2404:current"

var ValidLinuxImages = []string{
	// Ubuntu 20.04
	"ubuntu-2004:2024.11.1",
	"ubuntu-2004:2024.08.1",
	"ubuntu-2004:2024.05.1",
	"ubuntu-2004:2024.04.4",
	"ubuntu-2004:2024.01.2",
	"ubuntu-2004:2024.01.1",
	"ubuntu-2004:2023.10.1",
	"ubuntu-2004:2023.07.1",
	"ubuntu-2004:2023.04.2",
	"ubuntu-2004:2023.04.1",
	"ubuntu-2004:2023.02.1",
	"ubuntu-2004:2022.10.1",
	"ubuntu-2004:2022.07.1",
	"ubuntu-2004:2022.04.2",
	"ubuntu-2004:2022.04.1",
	"ubuntu-2004:202201-02",
	"ubuntu-2004:202201-01",
	"ubuntu-2004:202111-02",
	"ubuntu-2004:202111-01",
	"ubuntu-2004:202107-02",
	"ubuntu-2004:202104-01",
	"ubuntu-2004:202101-01",
	"ubuntu-2004:202010-01",
	"ubuntu-2004:current",
	"ubuntu-2004:edge",

	// Ubuntu 22.04
	"ubuntu-2204:2024.11.1",
	"ubuntu-2204:2024.08.1",
	"ubuntu-2204:2024.05.1",
	"ubuntu-2204:2024.04.4",
	"ubuntu-2204:2024.01.2",
	"ubuntu-2204:2024.01.1",
	"ubuntu-2204:2023.10.1",
	"ubuntu-2204:2023.07.2",
	"ubuntu-2204:2023.04.2",
	"ubuntu-2204:2023.04.1",
	"ubuntu-2204:2023.02.1",
	"ubuntu-2204:2022.10.2",
	"ubuntu-2204:2022.10.1",
	"ubuntu-2204:2022.07.2",
	"ubuntu-2204:2022.07.1",
	"ubuntu-2204:2022.04.2",
	"ubuntu-2204:2022.04.1",
	"ubuntu-2204:current",
	"ubuntu-2204:edge",

	// Ubuntu 24.04
	"ubuntu-2404:2024.11.1",
	"ubuntu-2404:2024.08.1",
	"ubuntu-2404:2024.05.1",
	"ubuntu-2404:current",
	"ubuntu-2404:edge",

	// Android
	"android:2024.11.1",
	"android:2024.04.1",
	"android:2024.01.1",
	"android:2023.11.1",
	"android:2023.10.1",
	"android:2023.09.1",
	"android:2023.08.1",
	"android:2023.07.1",
	"android:2023.06.1",
	"android:2023.05.1",
	"android:2023.04.1",
	"android:2023.03.1",
	"android:2023.02.1",
	"android:2022.12.1",
	"android:2022.09.1",
	"android:2022.08.1",
	"android:2022.07.1",
	"android:2022.06.2",
	"android:2022.06.1",
	"android:2022.04.1",
	"android:2022.03.1",
	"android:2022.01.1",
	"android:2021.12.1",
	"android:2021.10.1",
	"android:202102-01",
}

var ValidLinuxResourceClasses = []string{
	"small",
	"medium",
	"medium+",
	"large",
	"xlarge",
	"2xlarge",
	"2xlarge+",
	"arm.medium",
	"arm.large",
	"arm.xlarge",
	"arm.2xlarge",

	// Special case: missing resource class or empty string means default linux
	// resource class
	"",
}

var ValidWindowsImages = []string{
	// Windows Server 2019
	"windows-server-2019-vs2019:2024.12.1",
	"windows-server-2019-vs2019:2024.05.1",
	"windows-server-2019-vs2019:2024.01.1",
	"windows-server-2019-vs2019:2023.10.1",
	"windows-server-2019-vs2019:2023.08.1",
	"windows-server-2019-vs2019:2023.04.1",
	"windows-server-2019-vs2019:2022.08.1",
	"windows-server-2019-vs2019:current",
	"windows-server-2019-vs2019:edge",

	// Windows Server 2022
	"windows-server-2022-gui:2024.04.1",
	"windows-server-2022-gui:2024.01.1",
	"windows-server-2022-gui:2023.11.1",
	"windows-server-2022-gui:2023.10.1",
	"windows-server-2022-gui:2023.09.1",
	"windows-server-2022-gui:2023.08.1",
	"windows-server-2022-gui:2023.07.1",
	"windows-server-2022-gui:2023.06.1",
	"windows-server-2022-gui:2023.05.1",
	"windows-server-2022-gui:2023.04.1",
	"windows-server-2022-gui:2023.03.1",
	"windows-server-2022-gui:2022.08.1",
	"windows-server-2022-gui:2022.07.1",
	"windows-server-2022-gui:2022.06.1",
	"windows-server-2022-gui:2022.04.1",
	"windows-server-2022-gui:current",
	"windows-server-2022-gui:edge",
}

var ValidWindowsResourceClasses = []string{
	"windows.medium",
	"windows.large",
	"windows.xlarge",
	"windows.2xlarge",
}

var ValidLinuxGPUImages = []string{
	// CUDA 11
	"linux-cuda-11:default",
	"linux-cuda-11:edge",

	// CUDA 12
	"linux-cuda-12:default",
	"linux-cuda-12:edge",
}

var ValidLinuxGPUResourceClasses = []string{
	"gpu.nvidia.small.gen2",
	"gpu.nvidia.small.multi",
	"gpu.nvidia.medium.multi",
	"gpu.nvidia.medium",
	"gpu.nvidia.large",
}

var ValidWindowsGPUImages = []string{
	"windows-server-2019-cuda:current",
	"windows-server-2019-cuda:edge",
}

var ValidWindowsGPUResourceClasses = []string{
	"windows.gpu.nvidia.medium",
}

var ValidMachineResourceClasses = slices.Concat(
	ValidLinuxResourceClasses,
	ValidWindowsResourceClasses,
	ValidLinuxGPUResourceClasses,
	ValidWindowsGPUResourceClasses,
)

var ValidMachineImages = slices.Concat(
	ValidLinuxImages,
	ValidWindowsImages,
	ValidLinuxGPUImages,
	ValidWindowsGPUImages,
)

var ValidMachinePairs = []struct {
	Images          []string
	ResourceClasses []string
}{
	{Images: ValidLinuxImages, ResourceClasses: ValidLinuxResourceClasses},
	{Images: ValidWindowsImages, ResourceClasses: ValidWindowsResourceClasses},
	{Images: ValidLinuxGPUImages, ResourceClasses: ValidLinuxGPUResourceClasses},
	{Images: ValidWindowsGPUImages, ResourceClasses: ValidWindowsGPUResourceClasses},
}

var ValidXcodeVersions = []string{
	"16.0.0",
	"15.4.0",
	"15.3.0",
	"15.2.0",
	"15.1.0",
	"15.0.0",
	"14.3.1",
	"14.2.0",
	"14.1.0",
	"14.0.1",
	"13.4.1",
}

var ValidMacOSResourceClasses = []string{
	"macos.m1.medium.gen1",
	"macos.m1.large.gen1",
}

var ValidDockerResourceClasses = ValidLinuxResourceClasses

func IsSelfHostedRunner(resourceClass string) bool {
	return len(strings.Split(resourceClass, "/")) > 1
}
