package ast

import (
	"go.lsp.dev/protocol"
)

type Parameter interface {
	IsOptional() bool

	GetName() string
	GetNameRange() protocol.Range

	GetRange() protocol.Range

	GetType() string
	GetTypeRange() protocol.Range

	GetDefaultRange() protocol.Range

	GetDescription() string
}

type BaseParameter struct {
	Name         string
	NameRange    protocol.Range
	Range        protocol.Range
	HasDefault   bool
	Description  string
	TypeRange    protocol.Range
	DefaultRange protocol.Range
}

// String parameter definition
type StringParameter struct {
	BaseParameter
	Default string
}

func (p StringParameter) IsOptional() bool {
	return p.HasDefault
}

func (p StringParameter) GetName() string {
	return p.Name
}

func (p StringParameter) GetType() string {
	return "string"
}

func (p StringParameter) GetNameRange() protocol.Range {
	return p.NameRange
}

func (p StringParameter) GetRange() protocol.Range {
	return p.Range
}

func (p StringParameter) GetTypeRange() protocol.Range {
	return p.TypeRange
}

func (p StringParameter) GetDefaultRange() protocol.Range {
	return p.DefaultRange
}

func (p StringParameter) GetDescription() string {
	return p.Description
}

// Boolean parameter definition
type BooleanParameter struct {
	BaseParameter
	Default bool
}

func (p BooleanParameter) IsOptional() bool {
	return p.HasDefault
}

func (p BooleanParameter) GetName() string {
	return p.Name
}

func (p BooleanParameter) GetType() string {
	return "boolean"
}

func (p BooleanParameter) GetNameRange() protocol.Range {
	return p.NameRange
}

func (p BooleanParameter) GetRange() protocol.Range {
	return p.Range
}

func (p BooleanParameter) GetTypeRange() protocol.Range {
	return p.TypeRange
}

func (p BooleanParameter) GetDefaultRange() protocol.Range {
	return p.DefaultRange
}

func (p BooleanParameter) GetDescription() string {
	return p.Description
}

// Integer parameter definition
type IntegerParameter struct {
	BaseParameter
	Default int
}

func (p IntegerParameter) IsOptional() bool {
	return p.HasDefault
}

func (p IntegerParameter) GetName() string {
	return p.Name
}

func (p IntegerParameter) GetType() string {
	return "integer"
}

func (p IntegerParameter) GetNameRange() protocol.Range {
	return p.NameRange
}

func (p IntegerParameter) GetRange() protocol.Range {
	return p.Range
}

func (p IntegerParameter) GetTypeRange() protocol.Range {
	return p.TypeRange
}

func (p IntegerParameter) GetDefaultRange() protocol.Range {
	return p.DefaultRange
}

func (p IntegerParameter) GetDescription() string {
	return p.Description
}

// Enum parameter definition
type EnumParameter struct {
	BaseParameter
	Default string   // TODO: check
	Enum    []string // TODO: check
}

func (p EnumParameter) IsOptional() bool {
	return p.HasDefault
}

func (p EnumParameter) GetName() string {
	return p.Name
}

func (p EnumParameter) GetType() string {
	return "enum"
}

func (p EnumParameter) GetNameRange() protocol.Range {
	return p.NameRange
}

func (p EnumParameter) GetRange() protocol.Range {
	return p.Range
}

func (p EnumParameter) GetTypeRange() protocol.Range {
	return p.TypeRange
}

func (p EnumParameter) GetDefaultRange() protocol.Range {
	return p.DefaultRange
}

func (p EnumParameter) GetDescription() string {
	return p.Description
}

// Executor parameter definition
type ExecutorParameter struct {
	BaseParameter
	Default string
}

func (p ExecutorParameter) IsOptional() bool {
	return p.HasDefault
}

func (p ExecutorParameter) GetName() string {
	return p.Name
}

func (p ExecutorParameter) GetType() string {
	return "executor"
}

func (p ExecutorParameter) GetNameRange() protocol.Range {
	return p.NameRange
}

func (p ExecutorParameter) GetRange() protocol.Range {
	return p.Range
}

func (p ExecutorParameter) GetTypeRange() protocol.Range {
	return p.TypeRange
}

func (p ExecutorParameter) GetDefaultRange() protocol.Range {
	return p.DefaultRange
}

func (p ExecutorParameter) GetDescription() string {
	return p.Description
}

// Steps parameter definition
type StepsParameter struct {
	BaseParameter
	Default ParameterValue
}

func (p StepsParameter) IsOptional() bool {
	return p.HasDefault
}

func (p StepsParameter) GetName() string {
	return p.Name
}

func (p StepsParameter) GetType() string {
	return "steps"
}

func (p StepsParameter) GetNameRange() protocol.Range {
	return p.NameRange
}

func (p StepsParameter) GetRange() protocol.Range {
	return p.Range
}

func (p StepsParameter) GetTypeRange() protocol.Range {
	return p.TypeRange
}

func (p StepsParameter) GetDefaultRange() protocol.Range {
	return p.DefaultRange
}

func (p StepsParameter) GetDescription() string {
	return p.Description
}

// Environment Variable parameter definition
type EnvVariableParameter struct {
	BaseParameter
	Default string
}

func (p EnvVariableParameter) IsOptional() bool {
	return p.HasDefault
}

func (p EnvVariableParameter) GetName() string {
	return p.Name
}

func (p EnvVariableParameter) GetType() string {
	return "env_var_name"
}

func (p EnvVariableParameter) GetNameRange() protocol.Range {
	return p.NameRange
}

func (p EnvVariableParameter) GetRange() protocol.Range {
	return p.Range
}

func (p EnvVariableParameter) GetTypeRange() protocol.Range {
	return p.TypeRange
}

func (p EnvVariableParameter) GetDefaultRange() protocol.Range {
	return p.DefaultRange
}

func (p EnvVariableParameter) GetDescription() string {
	return p.Description
}
