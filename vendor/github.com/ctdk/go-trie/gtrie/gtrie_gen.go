package gtrie

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/philhofer/msgp)
// DO NOT EDIT

import (
	"github.com/philhofer/msgp/msgp"
)


// MarshalMsg implements the msgp.Marshaler interface
func (z *Transition) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())

	o = msgp.AppendMapHeader(o, 2)

	o = msgp.AppendString(o, "Child")

	if z.Child == nil {
		o = msgp.AppendNil(o)
	} else {

		o, err = z.Child.MarshalMsg(o)
		if err != nil {
			return
		}

	}

	o = msgp.AppendString(o, "Label")

	o = msgp.AppendInt32(o, int32(z.Label))

	return
}

// UnmarshalMsg unmarshals a Transition from MessagePack, returning any extra bytes
// and any errors encountered
func (z *Transition) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field

	var isz uint32
	isz, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for xplz := uint32(0); xplz < isz; xplz++ {
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

// Msgsize implements the msgp.Sizer interface
func (z *Transition) Msgsize() (s int) {

	s += msgp.MapHeaderSize
	s += msgp.StringPrefixSize + 5

	if z.Child == nil {
		s += msgp.NilSize
	} else {

		s += z.Child.Msgsize()

	}
	s += msgp.StringPrefixSize + 5

	s += msgp.Int32Size

	return
}

// DecodeMsg implements the msgp.Decodable interface
func (z *Transition) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field

	var isz uint32
	isz, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for xplz := uint32(0); xplz < isz; xplz++ {
		field, err = dc.ReadMapKey(field)
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

// EncodeMsg implements the msgp.Encodable interface
func (z *Transition) EncodeMsg(en *msgp.Writer) (err error) {

	err = en.WriteMapHeader(2)
	if err != nil {
		return
	}

	err = en.WriteString("Child")
	if err != nil {
		return
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

	err = en.WriteString("Label")
	if err != nil {
		return
	}

	err = en.WriteInt32(int32(z.Label))

	if err != nil {
		return
	}

	return
}

// MarshalMsg implements the msgp.Marshaler interface
func (z *Node) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())

	o = msgp.AppendMapHeader(o, 3)

	o = msgp.AppendString(o, "Id")

	o = msgp.AppendInt(o, int(z.Id))

	o = msgp.AppendString(o, "Terminal")

	o = msgp.AppendBool(o, z.Terminal)

	o = msgp.AppendString(o, "Transitions")

	o = msgp.AppendArrayHeader(o, uint32(len(z.Transitions)))
	for xvk := range z.Transitions {

		o, err = z.Transitions[xvk].MarshalMsg(o)
		if err != nil {
			return
		}

	}

	return
}

// UnmarshalMsg unmarshals a Node from MessagePack, returning any extra bytes
// and any errors encountered
func (z *Node) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field

	var isz uint32
	isz, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for xplz := uint32(0); xplz < isz; xplz++ {
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
			if cap(z.Transitions) >= int(xsz) {
				z.Transitions = z.Transitions[0:int(xsz)]
			} else {
				z.Transitions = make([]Transition, int(xsz))
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

// Msgsize implements the msgp.Sizer interface
func (z *Node) Msgsize() (s int) {

	s += msgp.MapHeaderSize
	s += msgp.StringPrefixSize + 2

	s += msgp.IntSize
	s += msgp.StringPrefixSize + 8

	s += msgp.BoolSize
	s += msgp.StringPrefixSize + 11

	s += msgp.ArrayHeaderSize
	for xvk := range z.Transitions {
		_ = xvk

		s += z.Transitions[xvk].Msgsize()

	}

	return
}

// DecodeMsg implements the msgp.Decodable interface
func (z *Node) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field

	var isz uint32
	isz, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for xplz := uint32(0); xplz < isz; xplz++ {
		field, err = dc.ReadMapKey(field)
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
			if cap(z.Transitions) > 0 && cap(z.Transitions) >= int(xsz) {
				z.Transitions = z.Transitions[0:int(xsz)]
			} else {
				z.Transitions = make([]Transition, int(xsz))
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

// EncodeMsg implements the msgp.Encodable interface
func (z *Node) EncodeMsg(en *msgp.Writer) (err error) {

	err = en.WriteMapHeader(3)
	if err != nil {
		return
	}

	err = en.WriteString("Id")
	if err != nil {
		return
	}

	err = en.WriteInt(int(z.Id))

	if err != nil {
		return
	}

	err = en.WriteString("Terminal")
	if err != nil {
		return
	}

	err = en.WriteBool(z.Terminal)

	if err != nil {
		return
	}

	err = en.WriteString("Transitions")
	if err != nil {
		return
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
