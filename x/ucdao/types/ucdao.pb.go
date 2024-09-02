// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: haqq/ucdao/v1/ucdao.proto

package types

import (
	fmt "fmt"
	_ "github.com/cosmos/cosmos-sdk/types/tx/amino"
	_ "github.com/cosmos/gogoproto/gogoproto"
	proto "github.com/cosmos/gogoproto/proto"
	io "io"
	math "math"
	math_bits "math/bits"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

// CollateralValueType defines the type of collateral value.
type CollateralValueType int32

const (
	// COLLATERAL_VALUE_TYPE_UNSPECIFIED is the unspecified collateral value type.
	CollateralValueType_COLLATERAL_VALUE_TYPE_UNSPECIFIED CollateralValueType = 0
	// COLLATERAL_VALUE_TYPE_STRICT is the strict collateral value type.
	CollateralValueType_COLLATERAL_VALUE_TYPE_STRICT CollateralValueType = 1
	// COLLATERAL_VALUE_TYPE_MASK is the mask collateral value type.
	CollateralValueType_COLLATERAL_VALUE_TYPE_MASK CollateralValueType = 2
)

var CollateralValueType_name = map[int32]string{
	0: "COLLATERAL_VALUE_TYPE_UNSPECIFIED",
	1: "COLLATERAL_VALUE_TYPE_STRICT",
	2: "COLLATERAL_VALUE_TYPE_MASK",
}

var CollateralValueType_value = map[string]int32{
	"COLLATERAL_VALUE_TYPE_UNSPECIFIED": 0,
	"COLLATERAL_VALUE_TYPE_STRICT":      1,
	"COLLATERAL_VALUE_TYPE_MASK":        2,
}

func (x CollateralValueType) String() string {
	return proto.EnumName(CollateralValueType_name, int32(x))
}

func (CollateralValueType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_d93b72bfaafec2b8, []int{0}
}

// Params defines the parameters for the dao module.
type Params struct {
	// enable_dao is the parameter to enable the module functionality.
	EnableDao bool `protobuf:"varint,1,opt,name=enable_dao,json=enableDao,proto3" json:"enable_dao,omitempty"`
	// allowed_collaterals is the allowed collateral values.
	AllowedCollaterals []*AllowedCollateral `protobuf:"bytes,2,rep,name=allowed_collaterals,json=allowedCollaterals,proto3" json:"allowed_collaterals,omitempty"`
}

func (m *Params) Reset()      { *m = Params{} }
func (*Params) ProtoMessage() {}
func (*Params) Descriptor() ([]byte, []int) {
	return fileDescriptor_d93b72bfaafec2b8, []int{0}
}
func (m *Params) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Params) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Params.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Params) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Params.Merge(m, src)
}
func (m *Params) XXX_Size() int {
	return m.Size()
}
func (m *Params) XXX_DiscardUnknown() {
	xxx_messageInfo_Params.DiscardUnknown(m)
}

var xxx_messageInfo_Params proto.InternalMessageInfo

func (m *Params) GetEnableDao() bool {
	if m != nil {
		return m.EnableDao
	}
	return false
}

func (m *Params) GetAllowedCollaterals() []*AllowedCollateral {
	if m != nil {
		return m.AllowedCollaterals
	}
	return nil
}

type AllowedCollateral struct {
	// value is the allowed collateral value.
	Value string `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
	// type is the allowed collateral value type.
	Type CollateralValueType `protobuf:"varint,2,opt,name=type,proto3,enum=haqq.ucdao.v1.CollateralValueType" json:"type,omitempty"`
}

func (m *AllowedCollateral) Reset()      { *m = AllowedCollateral{} }
func (*AllowedCollateral) ProtoMessage() {}
func (*AllowedCollateral) Descriptor() ([]byte, []int) {
	return fileDescriptor_d93b72bfaafec2b8, []int{1}
}
func (m *AllowedCollateral) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *AllowedCollateral) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_AllowedCollateral.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *AllowedCollateral) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AllowedCollateral.Merge(m, src)
}
func (m *AllowedCollateral) XXX_Size() int {
	return m.Size()
}
func (m *AllowedCollateral) XXX_DiscardUnknown() {
	xxx_messageInfo_AllowedCollateral.DiscardUnknown(m)
}

var xxx_messageInfo_AllowedCollateral proto.InternalMessageInfo

func (m *AllowedCollateral) GetValue() string {
	if m != nil {
		return m.Value
	}
	return ""
}

func (m *AllowedCollateral) GetType() CollateralValueType {
	if m != nil {
		return m.Type
	}
	return CollateralValueType_COLLATERAL_VALUE_TYPE_UNSPECIFIED
}

func init() {
	proto.RegisterEnum("haqq.ucdao.v1.CollateralValueType", CollateralValueType_name, CollateralValueType_value)
	proto.RegisterType((*Params)(nil), "haqq.ucdao.v1.Params")
	proto.RegisterType((*AllowedCollateral)(nil), "haqq.ucdao.v1.AllowedCollateral")
}

func init() { proto.RegisterFile("haqq/ucdao/v1/ucdao.proto", fileDescriptor_d93b72bfaafec2b8) }

var fileDescriptor_d93b72bfaafec2b8 = []byte{
	// 381 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x92, 0xcc, 0x48, 0x2c, 0x2c,
	0xd4, 0x2f, 0x4d, 0x4e, 0x49, 0xcc, 0xd7, 0x2f, 0x33, 0x84, 0x30, 0xf4, 0x0a, 0x8a, 0xf2, 0x4b,
	0xf2, 0x85, 0x78, 0x41, 0x52, 0x7a, 0x10, 0x91, 0x32, 0x43, 0x29, 0x91, 0xf4, 0xfc, 0xf4, 0x7c,
	0xb0, 0x8c, 0x3e, 0x88, 0x05, 0x51, 0x24, 0x25, 0x98, 0x98, 0x9b, 0x99, 0x97, 0xaf, 0x0f, 0x26,
	0x21, 0x42, 0x4a, 0x33, 0x18, 0xb9, 0xd8, 0x02, 0x12, 0x8b, 0x12, 0x73, 0x8b, 0x85, 0x64, 0xb9,
	0xb8, 0x52, 0xf3, 0x12, 0x93, 0x72, 0x52, 0xe3, 0x53, 0x12, 0xf3, 0x25, 0x18, 0x15, 0x18, 0x35,
	0x38, 0x82, 0x38, 0x21, 0x22, 0x2e, 0x89, 0xf9, 0x42, 0x81, 0x5c, 0xc2, 0x89, 0x39, 0x39, 0xf9,
	0xe5, 0xa9, 0x29, 0xf1, 0xc9, 0xf9, 0x39, 0x39, 0x89, 0x25, 0xa9, 0x45, 0x89, 0x39, 0xc5, 0x12,
	0x4c, 0x0a, 0xcc, 0x1a, 0xdc, 0x46, 0x0a, 0x7a, 0x28, 0xf6, 0xeb, 0x39, 0x42, 0x54, 0x3a, 0xc3,
	0x15, 0x06, 0x09, 0x25, 0xa2, 0x0b, 0x15, 0x5b, 0xc9, 0xcc, 0x58, 0x20, 0xcf, 0xd0, 0xf5, 0x7c,
	0x83, 0x96, 0x30, 0xd8, 0x63, 0x15, 0x50, 0xaf, 0x41, 0xdc, 0xa3, 0xd4, 0xc5, 0xc8, 0x25, 0x88,
	0x61, 0x8e, 0x90, 0x08, 0x17, 0x6b, 0x59, 0x62, 0x4e, 0x69, 0x2a, 0xd8, 0x81, 0x9c, 0x41, 0x10,
	0x8e, 0x90, 0x19, 0x17, 0x4b, 0x49, 0x65, 0x41, 0xaa, 0x04, 0x93, 0x02, 0xa3, 0x06, 0x9f, 0x91,
	0x12, 0x9a, 0x6b, 0x10, 0xda, 0xc3, 0x40, 0xaa, 0x43, 0x2a, 0x0b, 0x52, 0x83, 0xc0, 0xea, 0xad,
	0xd4, 0x61, 0x2e, 0x90, 0x43, 0x71, 0x01, 0x86, 0xb5, 0x5a, 0x75, 0x5c, 0xc2, 0x58, 0x4c, 0x11,
	0x52, 0xe5, 0x52, 0x74, 0xf6, 0xf7, 0xf1, 0x71, 0x0c, 0x71, 0x0d, 0x72, 0xf4, 0x89, 0x0f, 0x73,
	0xf4, 0x09, 0x75, 0x8d, 0x0f, 0x89, 0x0c, 0x70, 0x8d, 0x0f, 0xf5, 0x0b, 0x0e, 0x70, 0x75, 0xf6,
	0x74, 0xf3, 0x74, 0x75, 0x11, 0x60, 0x10, 0x52, 0xe0, 0x92, 0xc1, 0xae, 0x2c, 0x38, 0x24, 0xc8,
	0xd3, 0x39, 0x44, 0x80, 0x51, 0x48, 0x8e, 0x4b, 0x0a, 0xbb, 0x0a, 0x5f, 0xc7, 0x60, 0x6f, 0x01,
	0x26, 0x27, 0x97, 0x13, 0x8f, 0xe4, 0x18, 0x2f, 0x3c, 0x92, 0x63, 0x7c, 0xf0, 0x48, 0x8e, 0x71,
	0xc2, 0x63, 0x39, 0x86, 0x0b, 0x8f, 0xe5, 0x18, 0x6e, 0x3c, 0x96, 0x63, 0x88, 0xd2, 0x4a, 0xcf,
	0x2c, 0xc9, 0x28, 0x4d, 0xd2, 0x4b, 0xce, 0xcf, 0xd5, 0x07, 0x79, 0x42, 0x37, 0x2f, 0xb5, 0xa4,
	0x3c, 0xbf, 0x28, 0x5b, 0x1f, 0xc5, 0x47, 0x20, 0xdf, 0x16, 0x27, 0xb1, 0x81, 0x23, 0xdd, 0x18,
	0x10, 0x00, 0x00, 0xff, 0xff, 0xe4, 0x4c, 0x7c, 0x85, 0x49, 0x02, 0x00, 0x00,
}

func (m *Params) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Params) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Params) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.AllowedCollaterals) > 0 {
		for iNdEx := len(m.AllowedCollaterals) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.AllowedCollaterals[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintUcdao(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x12
		}
	}
	if m.EnableDao {
		i--
		if m.EnableDao {
			dAtA[i] = 1
		} else {
			dAtA[i] = 0
		}
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func (m *AllowedCollateral) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *AllowedCollateral) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *AllowedCollateral) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.Type != 0 {
		i = encodeVarintUcdao(dAtA, i, uint64(m.Type))
		i--
		dAtA[i] = 0x10
	}
	if len(m.Value) > 0 {
		i -= len(m.Value)
		copy(dAtA[i:], m.Value)
		i = encodeVarintUcdao(dAtA, i, uint64(len(m.Value)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintUcdao(dAtA []byte, offset int, v uint64) int {
	offset -= sovUcdao(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Params) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.EnableDao {
		n += 2
	}
	if len(m.AllowedCollaterals) > 0 {
		for _, e := range m.AllowedCollaterals {
			l = e.Size()
			n += 1 + l + sovUcdao(uint64(l))
		}
	}
	return n
}

func (m *AllowedCollateral) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Value)
	if l > 0 {
		n += 1 + l + sovUcdao(uint64(l))
	}
	if m.Type != 0 {
		n += 1 + sovUcdao(uint64(m.Type))
	}
	return n
}

func sovUcdao(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozUcdao(x uint64) (n int) {
	return sovUcdao(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Params) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowUcdao
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Params: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Params: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field EnableDao", wireType)
			}
			var v int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowUcdao
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				v |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.EnableDao = bool(v != 0)
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field AllowedCollaterals", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowUcdao
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthUcdao
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthUcdao
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.AllowedCollaterals = append(m.AllowedCollaterals, &AllowedCollateral{})
			if err := m.AllowedCollaterals[len(m.AllowedCollaterals)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipUcdao(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthUcdao
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *AllowedCollateral) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowUcdao
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: AllowedCollateral: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: AllowedCollateral: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Value", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowUcdao
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthUcdao
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthUcdao
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Value = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Type", wireType)
			}
			m.Type = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowUcdao
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Type |= CollateralValueType(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipUcdao(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthUcdao
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipUcdao(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowUcdao
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowUcdao
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowUcdao
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthUcdao
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupUcdao
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthUcdao
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthUcdao        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowUcdao          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupUcdao = fmt.Errorf("proto: unexpected end of group")
)