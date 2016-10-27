package gtrie

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import (
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *Node) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var isz uint32
	isz, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for isz > 0 {
		isz--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Id":
			{
				var tmp int
				tmp, err = dc.ReadInt()
				z.Id = NodeId(tmp)
			}
			if err != nil {
				return
			}
		case "Terminal":
			z.Terminal, err = dc.ReadBool()
			if err != nil {
				return
			}
		case "Transitions":
			var xsz uint32
			xsz, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Transitions) >= int(xsz) {
				z.Transitions = z.Transitions[:xsz]
			} else {
				z.Transitions = make([]Transition, xsz)
			}
			for xvk := range z.Transitions {
				err = z.Transitions[xvk].DecodeMsg(dc)
				if err != nil {
					return
				}
			}
		default:
			err = dc.Skip()
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *Node) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 3
	// write "Id"
	err = en.Append(0x83, 0xa2, 0x49, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteInt(int(z.Id))
	if err != nil {
		return
	}
	// write "Terminal"
	err = en.Append(0xa8, 0x54, 0x65, 0x72, 0x6d, 0x69, 0x6e, 0x61, 0x6c)
	if err != nil {
		return err
	}
	err = en.WriteBool(z.Terminal)
	if err != nil {
		return
	}
	// write "Transitions"
	err = en.Append(0xab, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Transitions)))
	if err != nil {
		return
	}
	for xvk := range z.Transitions {
		err = z.Transitions[xvk].EncodeMsg(en)
		if err != nil {
			return
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *Node) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 3
	// string "Id"
	o = append(o, 0x83, 0xa2, 0x49, 0x64)
	o = msgp.AppendInt(o, int(z.Id))
	// string "Terminal"
	o = append(o, 0xa8, 0x54, 0x65, 0x72, 0x6d, 0x69, 0x6e, 0x61, 0x6c)
	o = msgp.AppendBool(o, z.Terminal)
	// string "Transitions"
	o = append(o, 0xab, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Transitions)))
	for xvk := range z.Transitions {
		o, err = z.Transitions[xvk].MarshalMsg(o)
		if err != nil {
			return
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Node) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var isz uint32
	isz, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for isz > 0 {
		isz--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Id":
			{
				var tmp int
				tmp, bts, err = msgp.ReadIntBytes(bts)
				z.Id = NodeId(tmp)
			}
			if err != nil {
				return
			}
		case "Terminal":
			z.Terminal, bts, err = msgp.ReadBoolBytes(bts)
			if err != nil {
				return
			}
		case "Transitions":
			var xsz uint32
			xsz, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Transitions) >= int(xsz) {
				z.Transitions = z.Transitions[:xsz]
			} else {
				z.Transitions = make([]Transition, xsz)
			}
			for xvk := range z.Transitions {
				bts, err = z.Transitions[xvk].UnmarshalMsg(bts)
				if err != nil {
					return
				}
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

func (z *Node) Msgsize() (s int) {
	s = 1 + 3 + msgp.IntSize + 9 + msgp.BoolSize + 12 + msgp.ArrayHeaderSize
	for xvk := range z.Transitions {
		s += z.Transitions[xvk].Msgsize()
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *NodeId) DecodeMsg(dc *msgp.Reader) (err error) {
	{
		var tmp int
		tmp, err = dc.ReadInt()
		(*z) = NodeId(tmp)
	}
	if err != nil {
		return
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z NodeId) EncodeMsg(en *msgp.Writer) (err error) {
	err = en.WriteInt(int(z))
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z NodeId) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	o = msgp.AppendInt(o, int(z))
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *NodeId) UnmarshalMsg(bts []byte) (o []byte, err error) {
	{
		var tmp int
		tmp, bts, err = msgp.ReadIntBytes(bts)
		(*z) = NodeId(tmp)
	}
	if err != nil {
		return
	}
	o = bts
	return
}

func (z NodeId) Msgsize() (s int) {
	s = msgp.IntSize
	return
}

// DecodeMsg implements msgp.Decodable
func (z *Transition) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var isz uint32
	isz, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for isz > 0 {
		isz--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Child":
			if dc.IsNil() {
				err = dc.ReadNil()
				if err != nil {
					return
				}
				z.Child = nil
			} else {
				if z.Child == nil {
					z.Child = new(Node)
				}
				err = z.Child.DecodeMsg(dc)
				if err != nil {
					return
				}
			}
		case "Label":
			{
				var tmp int32
				tmp, err = dc.ReadInt32()
				z.Label = rune(tmp)
			}
			if err != nil {
				return
			}
		default:
			err = dc.Skip()
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *Transition) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 2
	// write "Child"
	err = en.Append(0x82, 0xa5, 0x43, 0x68, 0x69, 0x6c, 0x64)
	if err != nil {
		return err
	}
	if z.Child == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = z.Child.EncodeMsg(en)
		if err != nil {
			return
		}
	}
	// write "Label"
	err = en.Append(0xa5, 0x4c, 0x61, 0x62, 0x65, 0x6c)
	if err != nil {
		return err
	}
	err = en.WriteInt32(int32(z.Label))
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *Transition) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 2
	// string "Child"
	o = append(o, 0x82, 0xa5, 0x43, 0x68, 0x69, 0x6c, 0x64)
	if z.Child == nil {
		o = msgp.AppendNil(o)
	} else {
		o, err = z.Child.MarshalMsg(o)
		if err != nil {
			return
		}
	}
	// string "Label"
	o = append(o, 0xa5, 0x4c, 0x61, 0x62, 0x65, 0x6c)
	o = msgp.AppendInt32(o, int32(z.Label))
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Transition) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var isz uint32
	isz, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for isz > 0 {
		isz--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Child":
			if msgp.IsNil(bts) {
				bts, err = msgp.ReadNilBytes(bts)
				if err != nil {
					return
				}
				z.Child = nil
			} else {
				if z.Child == nil {
					z.Child = new(Node)
				}
				bts, err = z.Child.UnmarshalMsg(bts)
				if err != nil {
					return
				}
			}
		case "Label":
			{
				var tmp int32
				tmp, bts, err = msgp.ReadInt32Bytes(bts)
				z.Label = rune(tmp)
			}
			if err != nil {
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

func (z *Transition) Msgsize() (s int) {
	s = 1 + 6
	if z.Child == nil {
		s += msgp.NilSize
	} else {
		s += z.Child.Msgsize()
	}
	s += 6 + msgp.Int32Size
	return
}
