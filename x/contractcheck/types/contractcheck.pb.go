// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: haqq/contractcheck/v1/contractcheck.proto

package types

import (
	fmt "fmt"
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

// RegisterCoinProposal is a gov Content type to register a token pair for a
// native Cosmos coin.
type MintNFTProposal struct {
	// title of the proposal
	Title string `protobuf:"bytes,1,opt,name=title,proto3" json:"title,omitempty"`
	// description of the proposal
	Description string `protobuf:"bytes,2,opt,name=description,proto3" json:"description,omitempty"`
	// metadata slice of the native Cosmos coins
	ContractAddress string `protobuf:"bytes,3,opt,name=contract_address,json=contractAddress,proto3" json:"contract_address,omitempty"`
	// nft url
	NftUrl string `protobuf:"bytes,4,opt,name=nft_url,json=nftUrl,proto3" json:"nft_url,omitempty"`
	// destination address
	DestinationAddress string `protobuf:"bytes,5,opt,name=destination_address,json=destinationAddress,proto3" json:"destination_address,omitempty"`
}

func (m *MintNFTProposal) Reset()         { *m = MintNFTProposal{} }
func (m *MintNFTProposal) String() string { return proto.CompactTextString(m) }
func (*MintNFTProposal) ProtoMessage()    {}
func (*MintNFTProposal) Descriptor() ([]byte, []int) {
	return fileDescriptor_bfc850758b4be695, []int{0}
}
func (m *MintNFTProposal) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MintNFTProposal) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MintNFTProposal.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MintNFTProposal) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MintNFTProposal.Merge(m, src)
}
func (m *MintNFTProposal) XXX_Size() int {
	return m.Size()
}
func (m *MintNFTProposal) XXX_DiscardUnknown() {
	xxx_messageInfo_MintNFTProposal.DiscardUnknown(m)
}

var xxx_messageInfo_MintNFTProposal proto.InternalMessageInfo

func (m *MintNFTProposal) GetTitle() string {
	if m != nil {
		return m.Title
	}
	return ""
}

func (m *MintNFTProposal) GetDescription() string {
	if m != nil {
		return m.Description
	}
	return ""
}

func (m *MintNFTProposal) GetContractAddress() string {
	if m != nil {
		return m.ContractAddress
	}
	return ""
}

func (m *MintNFTProposal) GetNftUrl() string {
	if m != nil {
		return m.NftUrl
	}
	return ""
}

func (m *MintNFTProposal) GetDestinationAddress() string {
	if m != nil {
		return m.DestinationAddress
	}
	return ""
}

func init() {
	proto.RegisterType((*MintNFTProposal)(nil), "haqq.contractcheck.v1.MintNFTProposal")
}

func init() {
	proto.RegisterFile("haqq/contractcheck/v1/contractcheck.proto", fileDescriptor_bfc850758b4be695)
}

var fileDescriptor_bfc850758b4be695 = []byte{
	// 279 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xd2, 0xcc, 0x48, 0x2c, 0x2c,
	0xd4, 0x4f, 0xce, 0xcf, 0x2b, 0x29, 0x4a, 0x4c, 0x2e, 0x49, 0xce, 0x48, 0x4d, 0xce, 0xd6, 0x2f,
	0x33, 0x44, 0x15, 0xd0, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x12, 0x05, 0x29, 0xd5, 0x43, 0x95,
	0x29, 0x33, 0x94, 0x12, 0x49, 0xcf, 0x4f, 0xcf, 0x07, 0xab, 0xd0, 0x07, 0xb1, 0x20, 0x8a, 0x95,
	0x8e, 0x30, 0x72, 0xf1, 0xfb, 0x66, 0xe6, 0x95, 0xf8, 0xb9, 0x85, 0x04, 0x14, 0xe5, 0x17, 0xe4,
	0x17, 0x27, 0xe6, 0x08, 0x89, 0x70, 0xb1, 0x96, 0x64, 0x96, 0xe4, 0xa4, 0x4a, 0x30, 0x2a, 0x30,
	0x6a, 0x70, 0x06, 0x41, 0x38, 0x42, 0x0a, 0x5c, 0xdc, 0x29, 0xa9, 0xc5, 0xc9, 0x45, 0x99, 0x05,
	0x25, 0x99, 0xf9, 0x79, 0x12, 0x4c, 0x60, 0x39, 0x64, 0x21, 0x21, 0x4d, 0x2e, 0x01, 0x98, 0xad,
	0xf1, 0x89, 0x29, 0x29, 0x45, 0xa9, 0xc5, 0xc5, 0x12, 0xcc, 0x60, 0x65, 0xfc, 0x30, 0x71, 0x47,
	0x88, 0xb0, 0x90, 0x38, 0x17, 0x7b, 0x5e, 0x5a, 0x49, 0x7c, 0x69, 0x51, 0x8e, 0x04, 0x0b, 0x58,
	0x05, 0x5b, 0x5e, 0x5a, 0x49, 0x68, 0x51, 0x8e, 0x90, 0x3e, 0x97, 0x70, 0x4a, 0x6a, 0x71, 0x49,
	0x66, 0x5e, 0x22, 0xc8, 0x48, 0xb8, 0x31, 0xac, 0x60, 0x45, 0x42, 0x48, 0x52, 0x50, 0x93, 0xac,
	0x58, 0x5e, 0x2c, 0x90, 0x67, 0x70, 0xf2, 0x39, 0xf1, 0x48, 0x8e, 0xf1, 0xc2, 0x23, 0x39, 0xc6,
	0x07, 0x8f, 0xe4, 0x18, 0x27, 0x3c, 0x96, 0x63, 0xb8, 0xf0, 0x58, 0x8e, 0xe1, 0xc6, 0x63, 0x39,
	0x86, 0x28, 0xa3, 0xf4, 0xcc, 0x92, 0x8c, 0xd2, 0x24, 0xbd, 0xe4, 0xfc, 0x5c, 0x7d, 0x50, 0xc0,
	0xe8, 0xe6, 0xa5, 0x96, 0x94, 0xe7, 0x17, 0x65, 0x83, 0x39, 0xfa, 0x15, 0x68, 0x41, 0x5a, 0x52,
	0x59, 0x90, 0x5a, 0x9c, 0xc4, 0x06, 0x0e, 0x1b, 0x63, 0x40, 0x00, 0x00, 0x00, 0xff, 0xff, 0xb8,
	0xad, 0x20, 0x5f, 0x75, 0x01, 0x00, 0x00,
}

func (m *MintNFTProposal) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MintNFTProposal) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MintNFTProposal) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.DestinationAddress) > 0 {
		i -= len(m.DestinationAddress)
		copy(dAtA[i:], m.DestinationAddress)
		i = encodeVarintContractcheck(dAtA, i, uint64(len(m.DestinationAddress)))
		i--
		dAtA[i] = 0x2a
	}
	if len(m.NftUrl) > 0 {
		i -= len(m.NftUrl)
		copy(dAtA[i:], m.NftUrl)
		i = encodeVarintContractcheck(dAtA, i, uint64(len(m.NftUrl)))
		i--
		dAtA[i] = 0x22
	}
	if len(m.ContractAddress) > 0 {
		i -= len(m.ContractAddress)
		copy(dAtA[i:], m.ContractAddress)
		i = encodeVarintContractcheck(dAtA, i, uint64(len(m.ContractAddress)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.Description) > 0 {
		i -= len(m.Description)
		copy(dAtA[i:], m.Description)
		i = encodeVarintContractcheck(dAtA, i, uint64(len(m.Description)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Title) > 0 {
		i -= len(m.Title)
		copy(dAtA[i:], m.Title)
		i = encodeVarintContractcheck(dAtA, i, uint64(len(m.Title)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintContractcheck(dAtA []byte, offset int, v uint64) int {
	offset -= sovContractcheck(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *MintNFTProposal) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Title)
	if l > 0 {
		n += 1 + l + sovContractcheck(uint64(l))
	}
	l = len(m.Description)
	if l > 0 {
		n += 1 + l + sovContractcheck(uint64(l))
	}
	l = len(m.ContractAddress)
	if l > 0 {
		n += 1 + l + sovContractcheck(uint64(l))
	}
	l = len(m.NftUrl)
	if l > 0 {
		n += 1 + l + sovContractcheck(uint64(l))
	}
	l = len(m.DestinationAddress)
	if l > 0 {
		n += 1 + l + sovContractcheck(uint64(l))
	}
	return n
}

func sovContractcheck(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozContractcheck(x uint64) (n int) {
	return sovContractcheck(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *MintNFTProposal) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowContractcheck
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
			return fmt.Errorf("proto: MintNFTProposal: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MintNFTProposal: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Title", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowContractcheck
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
				return ErrInvalidLengthContractcheck
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthContractcheck
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Title = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Description", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowContractcheck
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
				return ErrInvalidLengthContractcheck
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthContractcheck
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Description = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ContractAddress", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowContractcheck
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
				return ErrInvalidLengthContractcheck
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthContractcheck
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ContractAddress = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field NftUrl", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowContractcheck
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
				return ErrInvalidLengthContractcheck
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthContractcheck
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.NftUrl = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field DestinationAddress", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowContractcheck
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
				return ErrInvalidLengthContractcheck
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthContractcheck
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.DestinationAddress = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipContractcheck(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthContractcheck
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
func skipContractcheck(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowContractcheck
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
					return 0, ErrIntOverflowContractcheck
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
					return 0, ErrIntOverflowContractcheck
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
				return 0, ErrInvalidLengthContractcheck
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupContractcheck
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthContractcheck
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthContractcheck        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowContractcheck          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupContractcheck = fmt.Errorf("proto: unexpected end of group")
)
