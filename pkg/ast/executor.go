package ast

import (
	"go.lsp.dev/protocol"
)

type Executor interface {
	GetRange() protocol.Range

	GetName() string
	GetNameRange() protocol.Range

	IsUncomplete() bool

	GetResourceClass() string

	GetParameters() map[string]Parameter
	GetParametersRange() protocol.Range

	GetEnvs() Environment
}

type BaseExecutor struct {
	Name                string
	NameRange           protocol.Range
	Range               protocol.Range
	ResourceClass       string
	ResourceClassRange  protocol.Range
	BuiltInParameters   ExecutableParameters
	UserParameters      map[string]Parameter
	UserParametersRange protocol.Range
	Uncomplete          bool
	Environment         Environment
}

type ExecutableParameters struct {
	Description      string
	Shell            string
	WorkingDirectory string
}

func (e BaseExecutor) GetRange() protocol.Range {
	return e.Range
}

func (e BaseExecutor) GetName() string {
	return e.Name
}

func (e BaseExecutor) GetNameRange() protocol.Range {
	return e.NameRange
}

func (e BaseExecutor) IsUncomplete() bool {
	return e.Uncomplete
}

func (e BaseExecutor) GetResourceClass() string {
	return e.ResourceClass
}

func (e BaseExecutor) GetParameters() map[string]Parameter {
	return e.UserParameters
}

func (e BaseExecutor) GetParametersRange() protocol.Range {
	return e.UserParametersRange
}

func (e BaseExecutor) GetEnvs() Environment {
	return e.Environment
}

type DockerExecutor struct {
	BaseExecutor
	Image         []DockerImage
	ServiceImages []DockerImage
}

func (e DockerExecutor) GetRange() protocol.Range {
	return e.Range
}

func (e DockerExecutor) GetName() string {
	return e.Name
}

func (e DockerExecutor) GetNameRange() protocol.Range {
	return e.NameRange
}

func (e DockerExecutor) IsUncomplete() bool {
	return e.Uncomplete
}

func (e DockerExecutor) GetResourceClass() string {
	return e.ResourceClass
}

func (e DockerExecutor) GetParameters() map[string]Parameter {
	return e.UserParameters
}

func (e DockerExecutor) GetParametersRange() protocol.Range {
	return e.UserParametersRange
}

func (e DockerExecutor) GetEnvs() Environment {
	return e.Environment
}

type DockerImage struct {
	Image      DockerImageInfo
	ImageRange protocol.Range

	Name        string
	Entrypoint  []string
	Command     []string
	User        string
	Environment map[string]string
	Auth        DockerImageAuth
	AwsAuth     DockerImageAWSAuth
}

type DockerImageInfo struct {
	Namespace string
	Name      string
	Tag       string

	FullPath string
}

type DockerImageAuth struct {
	Username string
	Password string
}

type DockerImageAWSAuth struct {
	AWSAccessKeyID     string
	AWSSecretAccessKey string
}

type MachineExecutor struct {
	BaseExecutor
	Image              string
	ImageRange         protocol.Range
	DockerLayerCaching bool
	Machine            bool
	IsDeprecated       bool // This field is true when using `machine: true`
}

func (e MachineExecutor) GetRange() protocol.Range {
	return e.Range
}

func (e MachineExecutor) GetName() string {
	return e.Name
}

func (e MachineExecutor) GetNameRange() protocol.Range {
	return e.NameRange
}

func (e MachineExecutor) IsUncomplete() bool {
	return e.Uncomplete
}

func (e MachineExecutor) GetResourceClass() string {
	return e.ResourceClass
}

func (e MachineExecutor) GetParameters() map[string]Parameter {
	return e.UserParameters
}

func (e MachineExecutor) GetParametersRange() protocol.Range {
	return e.UserParametersRange
}

func (e MachineExecutor) GetEnvs() Environment {
	return e.Environment
}

type MacOSExecutor struct {
	BaseExecutor
	Xcode      string
	XcodeRange protocol.Range
}

func (e MacOSExecutor) GetRange() protocol.Range {
	return e.Range
}

func (e MacOSExecutor) GetName() string {
	return e.Name
}

func (e MacOSExecutor) GetNameRange() protocol.Range {
	return e.NameRange
}

func (e MacOSExecutor) IsUncomplete() bool {
	return e.Uncomplete
}

func (e MacOSExecutor) GetResourceClass() string {
	return e.ResourceClass
}

func (e MacOSExecutor) GetParameters() map[string]Parameter {
	return e.UserParameters
}

func (e MacOSExecutor) GetParametersRange() protocol.Range {
	return e.UserParametersRange
}

func (e MacOSExecutor) GetEnvs() Environment {
	return e.Environment
}

type EnvironmentParameter map[string]string
