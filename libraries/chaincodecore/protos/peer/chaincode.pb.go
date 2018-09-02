// Code generated by protoc-gen-go. DO NOT EDIT.
// source: chaincode.proto

/*
Package peer is a generated protocol buffer package.

It is generated from these files:
	chaincode.proto
	chaincode_shim.proto
	response.proto

It has these top-level messages:
	ChaincodeID
	ChaincodeInput
	ChaincodeSpec
	ChaincodeDeploymentSpec
	ChaincodeInvocationSpec
	ChaincodeMessage
	PutStateInfo
	GetStateByRange
	GetQueryResult
	GetHistoryForKey
	QueryStateNext
	QueryStateClose
	QueryResultBytes
	QueryResponse
	Response
*/
package peer

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import google_protobuf "github.com/golang/protobuf/ptypes/timestamp"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// Confidentiality Levels
type ConfidentialityLevel int32

const (
	ConfidentialityLevel_PUBLIC       ConfidentialityLevel = 0
	ConfidentialityLevel_CONFIDENTIAL ConfidentialityLevel = 1
)

var ConfidentialityLevel_name = map[int32]string{
	0: "PUBLIC",
	1: "CONFIDENTIAL",
}
var ConfidentialityLevel_value = map[string]int32{
	"PUBLIC":       0,
	"CONFIDENTIAL": 1,
}

func (x ConfidentialityLevel) String() string {
	return proto.EnumName(ConfidentialityLevel_name, int32(x))
}
func (ConfidentialityLevel) EnumDescriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

type ChaincodeSpec_Type int32

const (
	ChaincodeSpec_UNDEFINED ChaincodeSpec_Type = 0
	ChaincodeSpec_GOLANG    ChaincodeSpec_Type = 1
	ChaincodeSpec_NODE      ChaincodeSpec_Type = 2
	ChaincodeSpec_CAR       ChaincodeSpec_Type = 3
	ChaincodeSpec_JAVA      ChaincodeSpec_Type = 4
)

var ChaincodeSpec_Type_name = map[int32]string{
	0: "UNDEFINED",
	1: "GOLANG",
	2: "NODE",
	3: "CAR",
	4: "JAVA",
}
var ChaincodeSpec_Type_value = map[string]int32{
	"UNDEFINED": 0,
	"GOLANG":    1,
	"NODE":      2,
	"CAR":       3,
	"JAVA":      4,
}

func (x ChaincodeSpec_Type) String() string {
	return proto.EnumName(ChaincodeSpec_Type_name, int32(x))
}
func (ChaincodeSpec_Type) EnumDescriptor() ([]byte, []int) { return fileDescriptor0, []int{2, 0} }

type ChaincodeDeploymentSpec_ExecutionEnvironment int32

const (
	ChaincodeDeploymentSpec_DOCKER ChaincodeDeploymentSpec_ExecutionEnvironment = 0
	ChaincodeDeploymentSpec_SYSTEM ChaincodeDeploymentSpec_ExecutionEnvironment = 1
)

var ChaincodeDeploymentSpec_ExecutionEnvironment_name = map[int32]string{
	0: "DOCKER",
	1: "SYSTEM",
}
var ChaincodeDeploymentSpec_ExecutionEnvironment_value = map[string]int32{
	"DOCKER": 0,
	"SYSTEM": 1,
}

func (x ChaincodeDeploymentSpec_ExecutionEnvironment) String() string {
	return proto.EnumName(ChaincodeDeploymentSpec_ExecutionEnvironment_name, int32(x))
}
func (ChaincodeDeploymentSpec_ExecutionEnvironment) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor0, []int{3, 0}
}

// ChaincodeID contains the path as specified by the deploy transaction
// that created it as well as the hashCode that is generated by the
// system for the path. From the user level (ie, CLI, REST API and so on)
// deploy transaction is expected to provide the path and other requests
// are expected to provide the hashCode. The other value will be ignored.
// Internally, the structure could contain both values. For instance, the
// hashCode will be set when first generated using the path
type ChaincodeID struct {
	// deploy transaction will use the path
	Path string `protobuf:"bytes,1,opt,name=path" json:"path,omitempty"`
	// all other requests will use the name (really a hashcode) generated by
	// the deploy transaction
	Name string `protobuf:"bytes,2,opt,name=name" json:"name,omitempty"`
	// user friendly version name for the chaincode
	Version string `protobuf:"bytes,3,opt,name=version" json:"version,omitempty"`
}

func (m *ChaincodeID) Reset()                    { *m = ChaincodeID{} }
func (m *ChaincodeID) String() string            { return proto.CompactTextString(m) }
func (*ChaincodeID) ProtoMessage()               {}
func (*ChaincodeID) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *ChaincodeID) GetPath() string {
	if m != nil {
		return m.Path
	}
	return ""
}

func (m *ChaincodeID) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *ChaincodeID) GetVersion() string {
	if m != nil {
		return m.Version
	}
	return ""
}

// Carries the chaincode function and its arguments.
// UnmarshalJSON in transaction.go converts the string-based REST/JSON input to
// the []byte-based current ChaincodeInput structure.
type ChaincodeInput struct {
	Args [][]byte `protobuf:"bytes,1,rep,name=args,proto3" json:"args,omitempty"`
}

func (m *ChaincodeInput) Reset()                    { *m = ChaincodeInput{} }
func (m *ChaincodeInput) String() string            { return proto.CompactTextString(m) }
func (*ChaincodeInput) ProtoMessage()               {}
func (*ChaincodeInput) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *ChaincodeInput) GetArgs() [][]byte {
	if m != nil {
		return m.Args
	}
	return nil
}

// Carries the chaincode specification. This is the actual metadata required for
// defining a chaincode.
type ChaincodeSpec struct {
	Type        ChaincodeSpec_Type `protobuf:"varint,1,opt,name=type,enum=protos.ChaincodeSpec_Type" json:"type,omitempty"`
	ChaincodeId *ChaincodeID       `protobuf:"bytes,2,opt,name=chaincode_id,json=chaincodeId" json:"chaincode_id,omitempty"`
	Input       *ChaincodeInput    `protobuf:"bytes,3,opt,name=input" json:"input,omitempty"`
	Timeout     int32              `protobuf:"varint,4,opt,name=timeout" json:"timeout,omitempty"`
}

func (m *ChaincodeSpec) Reset()                    { *m = ChaincodeSpec{} }
func (m *ChaincodeSpec) String() string            { return proto.CompactTextString(m) }
func (*ChaincodeSpec) ProtoMessage()               {}
func (*ChaincodeSpec) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

func (m *ChaincodeSpec) GetType() ChaincodeSpec_Type {
	if m != nil {
		return m.Type
	}
	return ChaincodeSpec_UNDEFINED
}

func (m *ChaincodeSpec) GetChaincodeId() *ChaincodeID {
	if m != nil {
		return m.ChaincodeId
	}
	return nil
}

func (m *ChaincodeSpec) GetInput() *ChaincodeInput {
	if m != nil {
		return m.Input
	}
	return nil
}

func (m *ChaincodeSpec) GetTimeout() int32 {
	if m != nil {
		return m.Timeout
	}
	return 0
}

// Specify the deployment of a chaincode.
// TODO: Define `codePackage`.
type ChaincodeDeploymentSpec struct {
	ChaincodeSpec *ChaincodeSpec `protobuf:"bytes,1,opt,name=chaincode_spec,json=chaincodeSpec" json:"chaincode_spec,omitempty"`
	// Controls when the chaincode becomes executable.
	EffectiveDate *google_protobuf.Timestamp                   `protobuf:"bytes,2,opt,name=effective_date,json=effectiveDate" json:"effective_date,omitempty"`
	CodePackage   []byte                                       `protobuf:"bytes,3,opt,name=code_package,json=codePackage,proto3" json:"code_package,omitempty"`
	ExecEnv       ChaincodeDeploymentSpec_ExecutionEnvironment `protobuf:"varint,4,opt,name=exec_env,json=execEnv,enum=protos.ChaincodeDeploymentSpec_ExecutionEnvironment" json:"exec_env,omitempty"`
}

func (m *ChaincodeDeploymentSpec) Reset()                    { *m = ChaincodeDeploymentSpec{} }
func (m *ChaincodeDeploymentSpec) String() string            { return proto.CompactTextString(m) }
func (*ChaincodeDeploymentSpec) ProtoMessage()               {}
func (*ChaincodeDeploymentSpec) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3} }

func (m *ChaincodeDeploymentSpec) GetChaincodeSpec() *ChaincodeSpec {
	if m != nil {
		return m.ChaincodeSpec
	}
	return nil
}

func (m *ChaincodeDeploymentSpec) GetEffectiveDate() *google_protobuf.Timestamp {
	if m != nil {
		return m.EffectiveDate
	}
	return nil
}

func (m *ChaincodeDeploymentSpec) GetCodePackage() []byte {
	if m != nil {
		return m.CodePackage
	}
	return nil
}

func (m *ChaincodeDeploymentSpec) GetExecEnv() ChaincodeDeploymentSpec_ExecutionEnvironment {
	if m != nil {
		return m.ExecEnv
	}
	return ChaincodeDeploymentSpec_DOCKER
}

// Carries the chaincode function and its arguments.
type ChaincodeInvocationSpec struct {
	ChaincodeSpec *ChaincodeSpec `protobuf:"bytes,1,opt,name=chaincode_spec,json=chaincodeSpec" json:"chaincode_spec,omitempty"`
	// This field can contain a user-specified ID generation algorithm
	// If supplied, this will be used to generate a ID
	// If not supplied (left empty), sha256base64 will be used
	// The algorithm consists of two parts:
	//  1, a hash function
	//  2, a decoding used to decode user (string) input to bytes
	// Currently, SHA256 with BASE64 is supported (e.g. idGenerationAlg='sha256base64')
	IdGenerationAlg string `protobuf:"bytes,2,opt,name=id_generation_alg,json=idGenerationAlg" json:"id_generation_alg,omitempty"`
}

func (m *ChaincodeInvocationSpec) Reset()                    { *m = ChaincodeInvocationSpec{} }
func (m *ChaincodeInvocationSpec) String() string            { return proto.CompactTextString(m) }
func (*ChaincodeInvocationSpec) ProtoMessage()               {}
func (*ChaincodeInvocationSpec) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{4} }

func (m *ChaincodeInvocationSpec) GetChaincodeSpec() *ChaincodeSpec {
	if m != nil {
		return m.ChaincodeSpec
	}
	return nil
}

func (m *ChaincodeInvocationSpec) GetIdGenerationAlg() string {
	if m != nil {
		return m.IdGenerationAlg
	}
	return ""
}

func init() {
	proto.RegisterType((*ChaincodeID)(nil), "protos.ChaincodeID")
	proto.RegisterType((*ChaincodeInput)(nil), "protos.ChaincodeInput")
	proto.RegisterType((*ChaincodeSpec)(nil), "protos.ChaincodeSpec")
	proto.RegisterType((*ChaincodeDeploymentSpec)(nil), "protos.ChaincodeDeploymentSpec")
	proto.RegisterType((*ChaincodeInvocationSpec)(nil), "protos.ChaincodeInvocationSpec")
	proto.RegisterEnum("protos.ConfidentialityLevel", ConfidentialityLevel_name, ConfidentialityLevel_value)
	proto.RegisterEnum("protos.ChaincodeSpec_Type", ChaincodeSpec_Type_name, ChaincodeSpec_Type_value)
	proto.RegisterEnum("protos.ChaincodeDeploymentSpec_ExecutionEnvironment", ChaincodeDeploymentSpec_ExecutionEnvironment_name, ChaincodeDeploymentSpec_ExecutionEnvironment_value)
}

func init() { proto.RegisterFile("chaincode.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 584 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xac, 0x54, 0x4d, 0x6f, 0xd3, 0x40,
	0x10, 0xad, 0x9b, 0xf4, 0x6b, 0xf2, 0x51, 0xb3, 0x14, 0x88, 0x7a, 0xa1, 0x58, 0x1c, 0xaa, 0x0a,
	0x1c, 0x29, 0x54, 0x9c, 0x10, 0x92, 0x1b, 0xbb, 0x95, 0x21, 0x24, 0x95, 0x9b, 0x22, 0xc1, 0x25,
	0xda, 0xd8, 0x13, 0x77, 0x85, 0xb3, 0x6b, 0xd9, 0x1b, 0xab, 0x39, 0xf3, 0xbf, 0xf8, 0x6b, 0xa0,
	0x5d, 0x37, 0x6e, 0xa3, 0xf6, 0xc8, 0x29, 0xbb, 0x6f, 0xdf, 0x9b, 0x79, 0xf3, 0x34, 0x31, 0xec,
	0x87, 0x37, 0x94, 0xf1, 0x50, 0x44, 0x68, 0xa7, 0x99, 0x90, 0x82, 0x6c, 0xeb, 0x9f, 0xfc, 0xf0,
	0x75, 0x2c, 0x44, 0x9c, 0x60, 0x57, 0x5f, 0xa7, 0x8b, 0x59, 0x57, 0xb2, 0x39, 0xe6, 0x92, 0xce,
	0xd3, 0x92, 0x68, 0x8d, 0xa0, 0xd1, 0x5f, 0x69, 0x7d, 0x97, 0x10, 0xa8, 0xa7, 0x54, 0xde, 0x74,
	0x8c, 0x23, 0xe3, 0x78, 0x2f, 0xd0, 0x67, 0x85, 0x71, 0x3a, 0xc7, 0xce, 0x66, 0x89, 0xa9, 0x33,
	0xe9, 0xc0, 0x4e, 0x81, 0x59, 0xce, 0x04, 0xef, 0xd4, 0x34, 0xbc, 0xba, 0x5a, 0x6f, 0xa1, 0x7d,
	0x5f, 0x90, 0xa7, 0x0b, 0xa9, 0xf4, 0x34, 0x8b, 0xf3, 0x8e, 0x71, 0x54, 0x3b, 0x6e, 0x06, 0xfa,
	0x6c, 0xfd, 0x35, 0xa0, 0x55, 0xd1, 0xae, 0x52, 0x0c, 0x89, 0x0d, 0x75, 0xb9, 0x4c, 0x51, 0x77,
	0x6e, 0xf7, 0x0e, 0x4b, 0x7b, 0xb9, 0xbd, 0x46, 0xb2, 0xc7, 0xcb, 0x14, 0x03, 0xcd, 0x23, 0x1f,
	0xa1, 0x59, 0x0d, 0x3d, 0x61, 0x91, 0x76, 0xd7, 0xe8, 0x3d, 0x7f, 0xa4, 0xf3, 0xdd, 0xa0, 0x51,
	0x11, 0xfd, 0x88, 0xbc, 0x83, 0x2d, 0xa6, 0x6c, 0x69, 0xdf, 0x8d, 0xde, 0xcb, 0xc7, 0x02, 0xf5,
	0x1a, 0x94, 0x24, 0x35, 0xa7, 0x4a, 0x4c, 0x2c, 0x64, 0xa7, 0x7e, 0x64, 0x1c, 0x6f, 0x05, 0xab,
	0xab, 0xf5, 0x19, 0xea, 0xca, 0x0d, 0x69, 0xc1, 0xde, 0xf5, 0xd0, 0xf5, 0xce, 0xfd, 0xa1, 0xe7,
	0x9a, 0x1b, 0x04, 0x60, 0xfb, 0x62, 0x34, 0x70, 0x86, 0x17, 0xa6, 0x41, 0x76, 0xa1, 0x3e, 0x1c,
	0xb9, 0x9e, 0xb9, 0x49, 0x76, 0xa0, 0xd6, 0x77, 0x02, 0xb3, 0xa6, 0xa0, 0x2f, 0xce, 0x77, 0xc7,
	0xac, 0x5b, 0x7f, 0x36, 0xe1, 0x55, 0xd5, 0xd3, 0xc5, 0x34, 0x11, 0xcb, 0x39, 0x72, 0xa9, 0xb3,
	0xf8, 0x04, 0xed, 0xfb, 0xd9, 0xf2, 0x14, 0x43, 0x9d, 0x4a, 0xa3, 0xf7, 0xe2, 0xc9, 0x54, 0x82,
	0x56, 0xb8, 0x96, 0xa4, 0x03, 0x6d, 0x9c, 0xcd, 0x30, 0x94, 0xac, 0xc0, 0x49, 0x44, 0x25, 0xde,
	0x65, 0x73, 0x68, 0x97, 0xcb, 0x60, 0xaf, 0x96, 0xc1, 0x1e, 0xaf, 0x96, 0x21, 0x68, 0x55, 0x0a,
	0x97, 0x4a, 0x24, 0x6f, 0xa0, 0xa9, 0x7b, 0xa7, 0x34, 0xfc, 0x45, 0x63, 0xd4, 0x59, 0x35, 0x83,
	0x86, 0xc2, 0x2e, 0x4b, 0x88, 0x8c, 0x60, 0x17, 0x6f, 0x31, 0x9c, 0x20, 0x2f, 0x74, 0x34, 0xed,
	0xde, 0xe9, 0x23, 0x77, 0xeb, 0x63, 0xd9, 0xde, 0x2d, 0x86, 0x0b, 0xc9, 0x04, 0xf7, 0x78, 0xc1,
	0x32, 0xc1, 0xd5, 0x43, 0xb0, 0xa3, 0xaa, 0x78, 0xbc, 0xb0, 0x6c, 0x38, 0x78, 0x8a, 0xa0, 0x12,
	0x75, 0x47, 0xfd, 0xaf, 0x5e, 0x50, 0xa6, 0x7b, 0xf5, 0xe3, 0x6a, 0xec, 0x7d, 0x33, 0x0d, 0xeb,
	0xb7, 0xf1, 0x20, 0x40, 0x9f, 0x17, 0x22, 0xa4, 0x4a, 0xfa, 0x1f, 0x02, 0x3c, 0x81, 0x67, 0x2c,
	0x9a, 0xc4, 0xc8, 0x31, 0xd3, 0x25, 0x27, 0x34, 0x89, 0xef, 0xb6, 0x7f, 0x9f, 0x45, 0x17, 0x15,
	0xee, 0x24, 0xf1, 0xc9, 0x29, 0x1c, 0xf4, 0x05, 0x9f, 0xb1, 0x08, 0xb9, 0x64, 0x34, 0x61, 0x72,
	0x39, 0xc0, 0x02, 0x13, 0xe5, 0xf4, 0xf2, 0xfa, 0x6c, 0xe0, 0xf7, 0xcd, 0x0d, 0x62, 0x42, 0xb3,
	0x3f, 0x1a, 0x9e, 0xfb, 0xae, 0x37, 0x1c, 0xfb, 0xce, 0xc0, 0x34, 0xce, 0xa6, 0xd0, 0x8d, 0xc5,
	0xfb, 0x05, 0x67, 0x32, 0x14, 0x8c, 0xdb, 0x09, 0x9b, 0x66, 0x34, 0x63, 0x98, 0xdb, 0x95, 0x91,
	0x50, 0x64, 0xb8, 0x32, 0x9b, 0x22, 0x66, 0x3f, 0x1f, 0x0a, 0xba, 0x95, 0xa0, 0xbb, 0x26, 0x28,
	0xff, 0xe6, 0x79, 0x57, 0x09, 0xa6, 0xe5, 0x27, 0xe0, 0xc3, 0xbf, 0x00, 0x00, 0x00, 0xff, 0xff,
	0x94, 0x29, 0x4a, 0x6c, 0x1c, 0x04, 0x00, 0x00,
}