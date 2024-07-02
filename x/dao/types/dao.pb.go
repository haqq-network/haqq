// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: haqq/dao/v1/dao.proto

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
	return fileDescriptor_c994d20d8e0ea63f, []int{0}
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
	return fileDescriptor_c994d20d8e0ea63f, []int{0}
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
	Type CollateralValueType `protobuf:"varint,2,opt,name=type,proto3,enum=haqq.dao.v1.CollateralValueType" json:"type,omitempty"`
}

func (m *AllowedCollateral) Reset()      { *m = AllowedCollateral{} }
func (*AllowedCollateral) ProtoMessage() {}
func (*AllowedCollateral) Descriptor() ([]byte, []int) {
	return fileDescriptor_c994d20d8e0ea63f, []int{1}
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
	proto.RegisterEnum("haqq.dao.v1.CollateralValueType", CollateralValueType_name, CollateralValueType_value)
	proto.RegisterType((*Params)(nil), "haqq.dao.v1.Params")
	proto.RegisterType((*AllowedCollateral)(nil), "haqq.dao.v1.AllowedCollateral")
}

func init() { proto.RegisterFile("haqq/dao/v1/dao.proto", fileDescriptor_c994d20d8e0ea63f) }

var fileDescriptor_c994d20d8e0ea63f = []byte{
	// 379 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x12, 0xcd, 0x48, 0x2c, 0x2c,
	0xd4, 0x4f, 0x49, 0xcc, 0xd7, 0x2f, 0x33, 0x04, 0x51, 0x7a, 0x05, 0x45, 0xf9, 0x25, 0xf9, 0x42,
	0xdc, 0x20, 0x61, 0x3d, 0x10, 0xbf, 0xcc, 0x50, 0x4a, 0x24, 0x3d, 0x3f, 0x3d, 0x1f, 0x2c, 0xae,
	0x0f, 0x62, 0x41, 0x94, 0x48, 0x09, 0x26, 0xe6, 0x66, 0xe6, 0xe5, 0xeb, 0x83, 0x49, 0x88, 0x90,
	0xd2, 0x14, 0x46, 0x2e, 0xb6, 0x80, 0xc4, 0xa2, 0xc4, 0xdc, 0x62, 0x21, 0x59, 0x2e, 0xae, 0xd4,
	0xbc, 0xc4, 0xa4, 0x9c, 0xd4, 0xf8, 0x94, 0xc4, 0x7c, 0x09, 0x46, 0x05, 0x46, 0x0d, 0x8e, 0x20,
	0x4e, 0x88, 0x88, 0x4b, 0x62, 0xbe, 0x90, 0x3f, 0x97, 0x70, 0x62, 0x4e, 0x4e, 0x7e, 0x79, 0x6a,
	0x4a, 0x7c, 0x72, 0x7e, 0x4e, 0x4e, 0x62, 0x49, 0x6a, 0x51, 0x62, 0x4e, 0xb1, 0x04, 0x93, 0x02,
	0xb3, 0x06, 0xb7, 0x91, 0x9c, 0x1e, 0x92, 0xed, 0x7a, 0x8e, 0x10, 0x75, 0xce, 0x70, 0x65, 0x41,
	0x42, 0x89, 0xe8, 0x42, 0xc5, 0x56, 0x52, 0x33, 0x16, 0xc8, 0x33, 0x74, 0x3d, 0xdf, 0xa0, 0x25,
	0x08, 0xf6, 0x50, 0x05, 0xd8, 0x4b, 0x10, 0xb7, 0x28, 0xb5, 0x31, 0x72, 0x09, 0x62, 0x98, 0x22,
	0x24, 0xc2, 0xc5, 0x5a, 0x96, 0x98, 0x53, 0x9a, 0x0a, 0x76, 0x1c, 0x67, 0x10, 0x84, 0x23, 0x64,
	0xc2, 0xc5, 0x52, 0x52, 0x59, 0x90, 0x2a, 0xc1, 0xa4, 0xc0, 0xa8, 0xc1, 0x67, 0xa4, 0x80, 0xe2,
	0x12, 0x84, 0xe6, 0x30, 0x90, 0xda, 0x90, 0xca, 0x82, 0xd4, 0x20, 0xb0, 0x6a, 0x2b, 0x55, 0x98,
	0xed, 0x32, 0x48, 0xb6, 0x63, 0x58, 0xa9, 0x55, 0xc7, 0x25, 0x8c, 0xc5, 0x0c, 0x21, 0x55, 0x2e,
	0x45, 0x67, 0x7f, 0x1f, 0x1f, 0xc7, 0x10, 0xd7, 0x20, 0x47, 0x9f, 0xf8, 0x30, 0x47, 0x9f, 0x50,
	0xd7, 0xf8, 0x90, 0xc8, 0x00, 0xd7, 0xf8, 0x50, 0xbf, 0xe0, 0x00, 0x57, 0x67, 0x4f, 0x37, 0x4f,
	0x57, 0x17, 0x01, 0x06, 0x21, 0x05, 0x2e, 0x19, 0xec, 0xca, 0x82, 0x43, 0x82, 0x3c, 0x9d, 0x43,
	0x04, 0x18, 0x85, 0xe4, 0xb8, 0xa4, 0xb0, 0xab, 0xf0, 0x75, 0x0c, 0xf6, 0x16, 0x60, 0x72, 0x72,
	0x3a, 0xf1, 0x48, 0x8e, 0xf1, 0xc2, 0x23, 0x39, 0xc6, 0x07, 0x8f, 0xe4, 0x18, 0x27, 0x3c, 0x96,
	0x63, 0xb8, 0xf0, 0x58, 0x8e, 0xe1, 0xc6, 0x63, 0x39, 0x86, 0x28, 0x8d, 0xf4, 0xcc, 0x92, 0x8c,
	0xd2, 0x24, 0xbd, 0xe4, 0xfc, 0x5c, 0x7d, 0x90, 0x17, 0x74, 0xf3, 0x52, 0x4b, 0xca, 0xf3, 0x8b,
	0xb2, 0xf5, 0x91, 0xfc, 0x03, 0xf2, 0x69, 0x71, 0x12, 0x1b, 0x38, 0xaa, 0x8d, 0x01, 0x01, 0x00,
	0x00, 0xff, 0xff, 0xe7, 0x56, 0xd1, 0x3e, 0x39, 0x02, 0x00, 0x00,
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
				i = encodeVarintDao(dAtA, i, uint64(size))
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
		i = encodeVarintDao(dAtA, i, uint64(m.Type))
		i--
		dAtA[i] = 0x10
	}
	if len(m.Value) > 0 {
		i -= len(m.Value)
		copy(dAtA[i:], m.Value)
		i = encodeVarintDao(dAtA, i, uint64(len(m.Value)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintDao(dAtA []byte, offset int, v uint64) int {
	offset -= sovDao(v)
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
			n += 1 + l + sovDao(uint64(l))
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
		n += 1 + l + sovDao(uint64(l))
	}
	if m.Type != 0 {
		n += 1 + sovDao(uint64(m.Type))
	}
	return n
}

func sovDao(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozDao(x uint64) (n int) {
	return sovDao(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Params) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowDao
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
					return ErrIntOverflowDao
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
					return ErrIntOverflowDao
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
				return ErrInvalidLengthDao
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthDao
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
			skippy, err := skipDao(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthDao
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
				return ErrIntOverflowDao
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
					return ErrIntOverflowDao
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
				return ErrInvalidLengthDao
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthDao
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
					return ErrIntOverflowDao
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
			skippy, err := skipDao(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthDao
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
func skipDao(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowDao
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
					return 0, ErrIntOverflowDao
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
					return 0, ErrIntOverflowDao
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
				return 0, ErrInvalidLengthDao
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupDao
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthDao
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthDao        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowDao          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupDao = fmt.Errorf("proto: unexpected end of group")
)