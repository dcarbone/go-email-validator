package emailvalidator

import (
	"errors"
	"fmt"
)

var (
	ErrUnexpectedNonGraphicCharacter   = errors.New("unexpected non-graphic ascii character seen")
	ErrUnexpectedCharacter             = errors.New("unexpected character seen")
	ErrInvalidUnquotedSequence         = errors.New("character sequence seen that requires quoting")
	ErrUnexpectedCharactersAfterDomain = fmt.Errorf("%w: after domain", ErrUnexpectedCharacter)
)

type ParseOptions struct {
	// todo: make this workable
	//AllowSmtpUtf8 bool

	// TrackCharacterPositions, if true, will cause the CharacterPositions map to be defined in the result
	TrackCharacterPositions bool
}

type OptFunc func(*ParseOptions)

func TrackCharacterPositions(opt *ParseOptions) {
	opt.TrackCharacterPositions = true
}

type Result struct {
	// Input is the verbatim provided value.
	Input string

	// Local contains the "local" portion of the email address, i.e. the part of the address prior to any sub-address
	// and the domain.
	Local string

	// Domain contains the "domain" portion of the email address, i.e. the part of the address after "@"
	Domain string

	// LiteralDomain will be true if the domain was an address-containing literal
	LiteralDomain bool

	// Comment may contain any seen comment in the address
	Comment string

	// Stripped will contain the email address minus any comment
	Stripped string

	// Quoted returns true if this email address was quoted
	Quoted bool

	// CharacterPositions contains the complete list of unique characters seen in this address, and the list of offsets
	// they were seen at.
	CharacterPositions map[string][]int

	// Err contains any / all errors seen during the validation of the address
	Err error
}

func BuildResult(email string, opts ...OptFunc) (Result, error) {
	const (
		strstr = "%s%s"
	)

	var (
		parseOpts ParseOptions
		chr       string
		dec       uint8
		prevDec   uint8
		nextDec   uint8
		err       error
		errs      []error

		inputLen = len(email)

		inLocal   = true
		inQuote   = false
		inComment = false
		inDomain  = false

		res = new(Result)
	)

	// set input verbatim
	res.Input = email

	// build options
	for _, fn := range opts {
		fn(&parseOpts)
	}

	// if we need to track character positions, do so.
	if parseOpts.TrackCharacterPositions {
		res.CharacterPositions = make(map[string][]int)
	}

	// iterate through provided value and do stuff.
	for i := 0; i < inputLen; i++ {

		// get current character and decimal in ascii table
		dec = email[i]
		chr = string(dec)

		// if we're beyond the first character, localize previous value
		if i > 0 {
			prevDec = email[i-1]
		}
		// if we've not reached the end, find the next character
		if i+1 < inputLen {
			nextDec = email[i+1]
		}

		// reset error
		err = nil

		// update char map, if configured to do so.
		if parseOpts.TrackCharacterPositions {
			if _, ok := res.CharacterPositions[chr]; !ok {
				res.CharacterPositions[chr] = make([]int, 0)
			}
			res.CharacterPositions[chr] = append(res.CharacterPositions[chr], i)
		}

		// make some decisions
		switch dec {

		case 0, // null
			1, // start of header
			2, // start of text
			3, // end of text
			4, // end of transmission
			5, // enquiry
			6, // ack
			7, // bell
			8: // backspace
			err = fmt.Errorf("%w: position %d", ErrUnexpectedNonGraphicCharacter, i)

		case 9: // horizontal tab
			// horizontal tab characters may only exist in the local portion of a quoted address
			if inDomain {
				err = fmt.Errorf("%w: horizontal tab at position %d in domain", ErrUnexpectedCharacter, i)
			} else if !inQuote {
				err = fmt.Errorf("%w: horizontal tab at position %d in local", ErrInvalidUnquotedSequence, i)
			}

		case 10, // LF
			11, // vertical tab
			12, // form feed
			13, // CR
			14, // shift out
			15, // shift in
			16, // data link escape
			17, // device control 1 (XON)
			18, // device control 2
			19, // device control 3 (XOFF)
			20, // device control 4
			21, // nack
			22, // synchronous idle
			23, // EOT
			24, // cancel
			25, // end of medium
			26, // substitute
			27, // esc
			28, // file separator
			29, // group separator
			30, // req to send / record separator
			31: // unit separator
			err = fmt.Errorf("%w: position %d", ErrUnexpectedNonGraphicCharacter, i)

		case 32: // space
			if inDomain {
				err = fmt.Errorf("%w: space at poosition %d in domain", ErrUnexpectedCharacter, i)
			} else if !inQuote && !inComment {
				err = fmt.Errorf("%w: space at position %d in local", ErrInvalidUnquotedSequence, i)
			}

		case 33: // !
			if inDomain {
				err = fmt.Errorf("%w: %q at position %d in domain", ErrUnexpectedCharacter, chr, i)
			}

		case 34: // "
			// the local portion of an address may contain one or more quoted sections
			if inLocal {
				if inQuote {
					// determine if this is an escaped quote
					if prevDec != 92 {
						//  if not escaped, mark sequence as ended and flip result quoted flag
						inQuote = false
						res.Quoted = true
					}
				} else {
					inQuote = true
				}
			} else {
				err = fmt.Errorf("%w: double quote at position %d", ErrUnexpectedCharacter, i)
			}

		case 35, // #
			36, // $
			37, // %
			38, // &
			39: // '
			if inDomain {
				err = fmt.Errorf("%w: %q at position %d in domain", ErrUnexpectedCharacter, chr, i)
			}

		case 40: // (
			// open parens are only allowed in quoted locals or as a comment opening marker
			if inDomain {
				err = fmt.Errorf("%w: %q at position %d in domain", ErrUnexpectedCharacter, chr, i)
			} else if inComment {
				err = fmt.Errorf("%w: %q at position %d in commment", ErrUnexpectedCharacter, chr, i)
			} else if !inQuote {
				inComment = true
			}

		case 41: // )
			// close parens are only allowed in quoted locals or as comment closing marker
			if inDomain {
				err = fmt.Errorf("%w: %q at position %d in domain", ErrUnexpectedCharacter, chr, i)
			} else if inComment {
				inComment = false
			} else if !inQuote {
				err = fmt.Errorf("%w: %q at position %d in local", ErrUnexpectedCharacter, chr, i)
			}

		case 42: // *
			// an astrix is only allowed in local portion
			if inDomain {
				err = fmt.Errorf("%w: %q at position %d in domain", ErrUnexpectedCharacter, chr, i)
			} else if inComment {
				err = fmt.Errorf("%w: %q at position %d in commment", ErrUnexpectedCharacter, chr, i)
			}

		case 43: // +
			// plus is only allowed in local, and may mark start of sub address
			if inDomain {
				err = fmt.Errorf("%w: %q at position %d in domain", ErrUnexpectedCharacter, chr, i)
			} else if inComment {
				err = fmt.Errorf("%w: %q at position %d in commment", ErrUnexpectedCharacter, chr, i)
			}

		case 44: // ,
			if inDomain {
				err = fmt.Errorf("%w: %q at position %d in domain", ErrUnexpectedCharacter, chr, i)
			} else if !inQuote {
				err = fmt.Errorf("%w: %q at position %d", ErrInvalidUnquotedSequence, chr, i)
			}

		case 45: // -
			// hyphen is only allowed in local, and may mark start of sub address
			if inDomain {
				err = fmt.Errorf("%w: %q at position %d in domain", ErrUnexpectedCharacter, chr, i)
			} else if inComment {
				err = fmt.Errorf("%w: %q at position %d in comment", ErrUnexpectedCharacter, chr, i)
			}

		case 46: // .
			if i == 0 {
				// period may not be the first character in the address local
				err = fmt.Errorf("%w: %q at position %d in local", ErrUnexpectedCharacter, chr, i)
			} else if prevDec == 46 {
				// if we're dealing with a double-dot sequence
				if inDomain {
					// not allowed at all in domain
					err = fmt.Errorf("%w: %q at position %d in domain", ErrUnexpectedCharacter, chr, i)
				} else if !inQuote {
					// only allowed in quoted local
					err = fmt.Errorf("%w: %q at position %d in local", ErrInvalidUnquotedSequence, chr, i)
				}
			} else if inComment {
				// not allowed in comments, maybe?
				err = fmt.Errorf("%w: %q at position %d in comment", ErrUnexpectedCharacter, chr, i)
			}

		case 47: // /
			if inDomain {
				err = fmt.Errorf("%w: %q at position %d in domain", ErrUnexpectedCharacter, chr, i)
			}

		case 48, // 0
			49, // 1
			50, // 2
			51, // 3
			52, // 4
			53, // 5
			54, // 6
			55, // 7
			56, // 8
			57: // 9

			// numbers allowed.

		case 58, // :
			59, // ;
			60: // <
			if inDomain {
				if !res.LiteralDomain {
					err = fmt.Errorf("%w: %q at position %d in domain", ErrUnexpectedCharacter, chr, i)
				}
			} else if inComment {
				// not allowed in comments?
				err = fmt.Errorf("%w: %q at position %d in comment", ErrUnexpectedCharacter, chr, i)
			} else if !inQuote {
				// must be in quoted sequence.
				err = fmt.Errorf("%w: %q at position %d in local", ErrInvalidUnquotedSequence, chr, i)
			}

		case 61: // =
			if inDomain {
				err = fmt.Errorf("%w: %q at position %d in domain", ErrUnexpectedCharacter, chr, i)
			}

		case 62: // >
			if inDomain {
				err = fmt.Errorf("%w: %q at position %d in domain", ErrUnexpectedCharacter, chr, i)
			} else if inComment {
				// not allowed in comments?
				err = fmt.Errorf("%w: %q at position %d in comment", ErrUnexpectedCharacter, chr, i)
			} else if !inQuote {
				// must be in quoted sequence.
				err = fmt.Errorf("%w: %q at position %d in local", ErrInvalidUnquotedSequence, chr, i)
			}

		case 63: // ?
			if inDomain {
				err = fmt.Errorf("%w: %q at position %d in domain", ErrUnexpectedCharacter, chr, i)
			}

		case 64: // @
			if inComment {
				// not allowed in comment
				err = fmt.Errorf("%w: %q at position %d in commment", ErrUnexpectedCharacter, chr, i)
			} else if inDomain {
				// not allowed in domain
				err = fmt.Errorf("%w: %q at position %d in domain", ErrUnexpectedCharacter, chr, i)
			} else if !inQuote {
				// if not in a quote sequence, end local sequence
				inLocal = false
				inDomain = true
			}

		case 65, // A
			66, // B
			67, // C
			68, // D
			69, // E
			70, // F
			71, // G
			72, // H
			73, // I
			74, // J
			75, // K
			76, // L
			77, // M
			78, // N
			79, // O
			80, // P
			81, // Q
			82, // R
			83, // S
			84, // T
			85, // U
			86, // V
			87, // W
			88, // X
			89, // Y
			90: // Z

			// upper alpha allowed.

		case 91: // [
			if inDomain {
				if len(res.Domain) == 0 {
					// mark beginning of literal domain sequence
					res.LiteralDomain = true
				} else {
					// not allowed at any other position
					err = fmt.Errorf("%w: %q at position %d in domain", ErrUnexpectedCharacter, chr, i)
				}
			} else if inComment {
				// not allowed in comments
				err = fmt.Errorf("%w: %q at position %d in comment", ErrUnexpectedCharacter, chr, i)
			} else if !inQuote {
				// only allowed in quotes
				err = fmt.Errorf("%w: %q at position %d in local", ErrInvalidUnquotedSequence, chr, i)
			}

		case 92: // \
			if inDomain {
				err = fmt.Errorf("%w: %q at position %d in domain", ErrUnexpectedCharacter, chr, i)
			} else if inComment {
				err = fmt.Errorf("%w: %q at position %d in comment", ErrUnexpectedCharacter, chr, i)
			} else if !inQuote {
				err = fmt.Errorf("%w: %q at position %d in local", ErrInvalidUnquotedSequence, chr, i)
			} else {
				switch nextDec {
				case 34, // "
					92: // \

					// these characters may be escaped through a backslash in a quoted sequence

				default:
					err = fmt.Errorf("%w: %q at position %d in local", ErrUnexpectedCharacter, chr, i)
				}
			}

		case 93: // ]
			if inDomain {
				if !res.LiteralDomain {
					err = fmt.Errorf("%w: %q at position %d in domain", ErrUnexpectedCharacter, chr, i)
				} else {
					inDomain = false
				}
			} else if inComment {
				// not allowed in comments
				err = fmt.Errorf("%w: %q at position %d in comment", ErrUnexpectedCharacter, chr, i)
			} else if !inQuote {
				// only allowed in quotes
				err = fmt.Errorf("%w: %q at position %d in local", ErrInvalidUnquotedSequence, chr, i)
			}

		case 94, // ^
			95, // _
			96: // `
			if inDomain {
				err = fmt.Errorf("%w: %q at position %d in domain", ErrUnexpectedCharacter, chr, i)
			}

		case 97, // a
			98,  // b
			99,  // c
			100, // d
			101, // e
			102, // f
			103, // g
			104, // h
			105, // i
			106, // j
			107, // k
			108, // l
			109, // m
			110, // n
			111, // o
			112, // p
			113, // q
			114, // r
			115, // s
			116, // t
			117, // u
			118, // v
			119, // w
			120, // x
			121, // y
			122: // z

			// lower alpha allowed.

		case 123, // {
			124, // |
			125, // }
			126: // ~
			if inDomain {
				err = fmt.Errorf("%w: %q at position %d in domain", ErrUnexpectedCharacter, chr, i)
			}

		case 127: // DEL
			err = fmt.Errorf("%w: position %d", ErrUnexpectedNonGraphicCharacter, i)

		default:
			err = fmt.Errorf("%w: position %d", ErrUnexpectedCharacter, i)
		}

		// if error, move on to next character
		if err != nil {
			// if error, add to error list.
			errs = append(errs, err)

			continue
		}

		// determine what to do with character

		if inLocal {
			// handle "local" portion

			if inComment {
				res.Comment = fmt.Sprintf(strstr, res.Comment, chr)
			} else {
				res.Local = fmt.Sprintf(strstr, res.Local, chr)
				res.Stripped = fmt.Sprintf(strstr, res.Stripped, chr)
			}
		} else if inDomain {
			if dec != 64 {
				res.Domain = fmt.Sprintf(strstr, res.Domain, chr)
			}
			res.Stripped = fmt.Sprintf(strstr, res.Stripped, chr)
		} else {
			errs = append(errs, fmt.Errorf("%w: %q at position %d beyond domain", ErrUnexpectedCharactersAfterDomain, chr, i))
		}
	}

	// return res and any errors seen.
	return *res, errors.Join(errs...)
}
