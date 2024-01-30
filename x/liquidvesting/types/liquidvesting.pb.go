// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: haqq/liquidvesting/v1/liquidvesting.proto

package types

import (
	fmt "fmt"
	_ "github.com/cosmos/cosmos-sdk/types"
	_ "github.com/cosmos/cosmos-sdk/types/tx/amino"
	github_com_cosmos_cosmos_sdk_x_auth_vesting_types "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	types "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	_ "github.com/cosmos/gogoproto/gogoproto"
	proto "github.com/cosmos/gogoproto/proto"
	github_com_cosmos_gogoproto_types "github.com/cosmos/gogoproto/types"
	_ "google.golang.org/protobuf/types/known/timestamppb"
	io "io"
	math "math"
	math_bits "math/bits"
	time "time"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf
var _ = time.Kitchen

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

// Denom represents liquid token bonded to some specific vesting schedule
type Denom struct {
	DenomName0    string `protobuf:"bytes,1,opt,name=denomName0,proto3" json:"denomName0,omitempty"`
	DenomName18   string `protobuf:"bytes,2,opt,name=denomName18,proto3" json:"denomName18,omitempty"`
	OriginalDenom string `protobuf:"bytes,3,opt,name=originalDenom,proto3" json:"originalDenom,omitempty"`
	// start date
	StartTime time.Time `protobuf:"bytes,4,opt,name=start_time,json=startTime,proto3,stdtime" json:"start_time"`
	// end_date
	EndTime time.Time `protobuf:"bytes,5,opt,name=end_time,json=endTime,proto3,stdtime" json:"end_time"`
	// lockup periods
	LockupPeriods github_com_cosmos_cosmos_sdk_x_auth_vesting_types.Periods `protobuf:"bytes,6,rep,name=lockup_periods,json=lockupPeriods,proto3,castrepeated=github.com/cosmos/cosmos-sdk/x/auth/vesting/types.Periods" json:"lockup_periods"`
}

func (m *Denom) Reset()         { *m = Denom{} }
func (m *Denom) String() string { return proto.CompactTextString(m) }
func (*Denom) ProtoMessage()    {}
func (*Denom) Descriptor() ([]byte, []int) {
	return fileDescriptor_ce2378517a6b5c6c, []int{0}
}
func (m *Denom) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Denom) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Denom.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Denom) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Denom.Merge(m, src)
}
func (m *Denom) XXX_Size() int {
	return m.Size()
}
func (m *Denom) XXX_DiscardUnknown() {
	xxx_messageInfo_Denom.DiscardUnknown(m)
}

var xxx_messageInfo_Denom proto.InternalMessageInfo

func (m *Denom) GetDenomName0() string {
	if m != nil {
		return m.DenomName0
	}
	return ""
}

func (m *Denom) GetDenomName18() string {
	if m != nil {
		return m.DenomName18
	}
	return ""
}

func (m *Denom) GetOriginalDenom() string {
	if m != nil {
		return m.OriginalDenom
	}
	return ""
}

func (m *Denom) GetStartTime() time.Time {
	if m != nil {
		return m.StartTime
	}
	return time.Time{}
}

func (m *Denom) GetEndTime() time.Time {
	if m != nil {
		return m.EndTime
	}
	return time.Time{}
}

func (m *Denom) GetLockupPeriods() github_com_cosmos_cosmos_sdk_x_auth_vesting_types.Periods {
	if m != nil {
		return m.LockupPeriods
	}
	return nil
}

func init() {
	proto.RegisterType((*Denom)(nil), "haqq.liquidvesting.Denom")
}

func init() {
	proto.RegisterFile("haqq/liquidvesting/v1/liquidvesting.proto", fileDescriptor_ce2378517a6b5c6c)
}

var fileDescriptor_ce2378517a6b5c6c = []byte{
	// 408 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x94, 0x52, 0xb1, 0xae, 0xd3, 0x30,
	0x14, 0x8d, 0x29, 0xef, 0xf1, 0x9e, 0xab, 0x87, 0x44, 0xc4, 0x10, 0x65, 0x70, 0x22, 0xf4, 0x86,
	0x32, 0x3c, 0x9b, 0x96, 0x05, 0x26, 0x44, 0x61, 0x44, 0x08, 0x55, 0x4c, 0x2c, 0x95, 0x93, 0x98,
	0xd4, 0x6a, 0xe2, 0x9b, 0xc6, 0x4e, 0x29, 0x1b, 0x23, 0x63, 0xbf, 0x83, 0x2f, 0xe9, 0xd8, 0x91,
	0x89, 0xa2, 0xf6, 0x47, 0x50, 0x9c, 0xb4, 0x34, 0x6c, 0x2c, 0xc9, 0xbd, 0xe7, 0x9e, 0x9c, 0x93,
	0x7b, 0x6c, 0xfc, 0x74, 0xc6, 0x17, 0x0b, 0x96, 0xc9, 0x45, 0x25, 0x93, 0xa5, 0xd0, 0x46, 0xaa,
	0x94, 0x2d, 0x87, 0x5d, 0x80, 0x16, 0x25, 0x18, 0x70, 0xdd, 0x9a, 0x4a, 0x3b, 0x13, 0xff, 0x11,
	0xcf, 0xa5, 0x02, 0x66, 0x9f, 0x0d, 0xcd, 0x7f, 0x9c, 0x42, 0x0a, 0xb6, 0x64, 0x75, 0xd5, 0xa2,
	0xb7, 0x31, 0xe8, 0x1c, 0x34, 0xfb, 0xeb, 0x11, 0x09, 0xc3, 0x87, 0xac, 0x63, 0xe1, 0x07, 0x29,
	0x40, 0x9a, 0x09, 0x66, 0xbb, 0xa8, 0xfa, 0xcc, 0x8c, 0xcc, 0x85, 0x36, 0x3c, 0x2f, 0x5a, 0x02,
	0x69, 0x65, 0x22, 0xae, 0xc5, 0x49, 0x23, 0x06, 0xa9, 0x9a, 0xf9, 0x93, 0x6f, 0x3d, 0x7c, 0xf1,
	0x56, 0x28, 0xc8, 0x5d, 0x82, 0x71, 0x52, 0x17, 0xef, 0x79, 0x2e, 0x9e, 0x79, 0x28, 0x44, 0x83,
	0xeb, 0xc9, 0x19, 0xe2, 0x86, 0xb8, 0x7f, 0xea, 0x86, 0x2f, 0xbc, 0x7b, 0x96, 0x70, 0x0e, 0xb9,
	0xb7, 0xf8, 0x06, 0x4a, 0x99, 0x4a, 0xc5, 0x33, 0x2b, 0xe9, 0xf5, 0x2c, 0xa7, 0x0b, 0xba, 0x6f,
	0x30, 0xd6, 0x86, 0x97, 0x66, 0x5a, 0xff, 0xaa, 0x77, 0x3f, 0x44, 0x83, 0xfe, 0xc8, 0xa7, 0xcd,
	0x1e, 0xf4, 0xb8, 0x07, 0xfd, 0x78, 0xdc, 0x63, 0x7c, 0xb5, 0xf9, 0x15, 0x38, 0xeb, 0x5d, 0x80,
	0x26, 0xd7, 0xf6, 0xbb, 0x7a, 0xe2, 0xbe, 0xc2, 0x57, 0x42, 0x25, 0x8d, 0xc4, 0xc5, 0x7f, 0x48,
	0x3c, 0x10, 0x2a, 0xb1, 0x02, 0xdf, 0x11, 0x7e, 0x98, 0x41, 0x3c, 0xaf, 0x8a, 0x69, 0x21, 0x4a,
	0x09, 0x89, 0xf6, 0x2e, 0xc3, 0xde, 0xa0, 0x3f, 0x22, 0xb4, 0x49, 0x8c, 0x1e, 0x83, 0x6e, 0x43,
	0xa3, 0x1f, 0x2c, 0x6d, 0xfc, 0xba, 0xd6, 0xfa, 0xb1, 0x0b, 0x5e, 0xa6, 0xd2, 0xcc, 0xaa, 0x88,
	0xc6, 0x90, 0xb3, 0x36, 0xe3, 0xe6, 0x75, 0xa7, 0x93, 0x39, 0x5b, 0x31, 0x5e, 0x99, 0xd9, 0xe9,
	0xf0, 0xcc, 0xd7, 0x42, 0xe8, 0x56, 0x41, 0x4f, 0x6e, 0x1a, 0xe3, 0xb6, 0x1d, 0xbf, 0xdb, 0xec,
	0x09, 0xda, 0xee, 0x09, 0xfa, 0xbd, 0x27, 0x68, 0x7d, 0x20, 0xce, 0xf6, 0x40, 0x9c, 0x9f, 0x07,
	0xe2, 0x7c, 0x1a, 0x9d, 0x79, 0xd4, 0x77, 0xe9, 0x4e, 0x09, 0xf3, 0x05, 0xca, 0xb9, 0x6d, 0xd8,
	0xea, 0x9f, 0x5b, 0x68, 0x4d, 0xa2, 0x4b, 0xbb, 0xff, 0xf3, 0x3f, 0x01, 0x00, 0x00, 0xff, 0xff,
	0x30, 0xe6, 0x95, 0x4c, 0xa8, 0x02, 0x00, 0x00,
}

func (m *Denom) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Denom) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Denom) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.LockupPeriods) > 0 {
		for iNdEx := len(m.LockupPeriods) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.LockupPeriods[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintLiquidvesting(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x32
		}
	}
	n1, err1 := github_com_cosmos_gogoproto_types.StdTimeMarshalTo(m.EndTime, dAtA[i-github_com_cosmos_gogoproto_types.SizeOfStdTime(m.EndTime):])
	if err1 != nil {
		return 0, err1
	}
	i -= n1
	i = encodeVarintLiquidvesting(dAtA, i, uint64(n1))
	i--
	dAtA[i] = 0x2a
	n2, err2 := github_com_cosmos_gogoproto_types.StdTimeMarshalTo(m.StartTime, dAtA[i-github_com_cosmos_gogoproto_types.SizeOfStdTime(m.StartTime):])
	if err2 != nil {
		return 0, err2
	}
	i -= n2
	i = encodeVarintLiquidvesting(dAtA, i, uint64(n2))
	i--
	dAtA[i] = 0x22
	if len(m.OriginalDenom) > 0 {
		i -= len(m.OriginalDenom)
		copy(dAtA[i:], m.OriginalDenom)
		i = encodeVarintLiquidvesting(dAtA, i, uint64(len(m.OriginalDenom)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.DenomName18) > 0 {
		i -= len(m.DenomName18)
		copy(dAtA[i:], m.DenomName18)
		i = encodeVarintLiquidvesting(dAtA, i, uint64(len(m.DenomName18)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.DenomName0) > 0 {
		i -= len(m.DenomName0)
		copy(dAtA[i:], m.DenomName0)
		i = encodeVarintLiquidvesting(dAtA, i, uint64(len(m.DenomName0)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintLiquidvesting(dAtA []byte, offset int, v uint64) int {
	offset -= sovLiquidvesting(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Denom) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.DenomName0)
	if l > 0 {
		n += 1 + l + sovLiquidvesting(uint64(l))
	}
	l = len(m.DenomName18)
	if l > 0 {
		n += 1 + l + sovLiquidvesting(uint64(l))
	}
	l = len(m.OriginalDenom)
	if l > 0 {
		n += 1 + l + sovLiquidvesting(uint64(l))
	}
	l = github_com_cosmos_gogoproto_types.SizeOfStdTime(m.StartTime)
	n += 1 + l + sovLiquidvesting(uint64(l))
	l = github_com_cosmos_gogoproto_types.SizeOfStdTime(m.EndTime)
	n += 1 + l + sovLiquidvesting(uint64(l))
	if len(m.LockupPeriods) > 0 {
		for _, e := range m.LockupPeriods {
			l = e.Size()
			n += 1 + l + sovLiquidvesting(uint64(l))
		}
	}
	return n
}

func sovLiquidvesting(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozLiquidvesting(x uint64) (n int) {
	return sovLiquidvesting(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Denom) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowLiquidvesting
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
			return fmt.Errorf("proto: Denom: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Denom: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field DenomName0", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowLiquidvesting
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
				return ErrInvalidLengthLiquidvesting
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthLiquidvesting
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.DenomName0 = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field DenomName18", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowLiquidvesting
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
				return ErrInvalidLengthLiquidvesting
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthLiquidvesting
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.DenomName18 = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field OriginalDenom", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowLiquidvesting
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
				return ErrInvalidLengthLiquidvesting
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthLiquidvesting
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.OriginalDenom = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field StartTime", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowLiquidvesting
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
				return ErrInvalidLengthLiquidvesting
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthLiquidvesting
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := github_com_cosmos_gogoproto_types.StdTimeUnmarshal(&m.StartTime, dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field EndTime", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowLiquidvesting
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
				return ErrInvalidLengthLiquidvesting
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthLiquidvesting
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := github_com_cosmos_gogoproto_types.StdTimeUnmarshal(&m.EndTime, dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field LockupPeriods", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowLiquidvesting
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
				return ErrInvalidLengthLiquidvesting
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthLiquidvesting
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.LockupPeriods = append(m.LockupPeriods, types.Period{})
			if err := m.LockupPeriods[len(m.LockupPeriods)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipLiquidvesting(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthLiquidvesting
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
func skipLiquidvesting(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowLiquidvesting
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
					return 0, ErrIntOverflowLiquidvesting
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
					return 0, ErrIntOverflowLiquidvesting
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
				return 0, ErrInvalidLengthLiquidvesting
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupLiquidvesting
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthLiquidvesting
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthLiquidvesting        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowLiquidvesting          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupLiquidvesting = fmt.Errorf("proto: unexpected end of group")
)
