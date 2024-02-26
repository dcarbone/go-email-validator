package emailvalidator_test

import (
	"errors"
	"testing"

	emailvalidator "github.com/dcarbone/go-email-validator"
)

type testStep struct {
	label string
	input string
	err   error
}

func TestBuildResult(t *testing.T) {
	// steps shamelessly stolen from:
	// https://en.wikipedia.org/wiki/Email_address#Examples

	steps := []testStep{

		// should produce no error

		{
			label: "simple",
			input: "simple@example.com",
		},
		{
			label: "very-common",
			input: "very.common@example.com",
		},
		{
			label: "one-letter-local",
			input: "x@example.com",
		},
		{
			label: "long-address-with-hyphens-and-subdomains",
			input: "long.email-address-with-hyphens@and.subdomains.example.com",
		},
		{
			label: "username-tag-sorting",
			input: "user.name+tag+sorting@example.com",
		},
		{
			label: "slashes",
			input: "name/surname@example.com",
		},
		{
			label: "dotless",
			input: "admin@example",
		},
		{
			label: "example-tld",
			input: "example@s.example",
		},
		{
			label: "quoted-space",
			input: `" "@example.org`,
		},
		{
			label: "quoted-double-dot",
			input: `"john..doe"@example.org`,
		},
		{
			label: "bangfield",
			input: "mailhost!username@example.org",
		},
		{
			label: "buncha-weirdness",
			input: `"very.(),:;<>[]\".VERY.\"very@\\ \"very\".unusual"@strange.example.com`,
		},
		{
			label: "escaped-mail-route",
			input: "user%example.com@example.org",
		},
		{
			label: "trailing-hyphen",
			input: "user-@example.org",
		},
		{
			label: "literal-domain-ipv4",
			input: "postmaster@[123.123.123.123]",
		},
		{
			label: "literal-domain-ipv6",
			input: "postmaster@[IPv6:2001:0db8:85a3:0000:0000:8a2e:0370:7334]",
		},
		{
			label: "underscore-prefixed-literal-domain-ipv6",
			input: "postmaster@[IPv6:2001:0db8:85a3:0000:0000:8a2e:0370:7334]",
		},

		// should produce error

		{
			label: "missing-local",
			input: "abc.example.com",
			err:   errors.New("FIXME"),
		},
		{
			label: "unquoted-at-in-local",
			input: "a@b@c@example.com",
			err:   emailvalidator.ErrUnexpectedCharacter,
		},
		{
			label: "unquoted-specials",
			input: `a"b(c)d,e:f;g<h>i[j\k]l@example.com`,
			err:   emailvalidator.ErrUnexpectedCharacter,
		},
		{
			label: "un-dotted-quotes",
			input: `just"not"right@example.com`,
			err:   emailvalidator.ErrUnexpectedCharacter,
		},
		{
			label: "unquoted-space-backslash",
			input: `this is"not\allowed@example.com`,
			err:   emailvalidator.ErrUnexpectedCharacter,
		},
		{
			label: "additional-quoted-badness",
			input: `this\ still\"not\\allowed@example.com`,
			err:   emailvalidator.ErrUnexpectedCharacter,
		},
		{
			label: "bigass-local",
			input: "1234567890123456789012345678901234567890123456789012345678901234+x@example.com",
			err:   emailvalidator.ErrUnexpectedCharacter,
		},
		{
			label: "underscores-in-domain",
			input: "i.like.underscores@but_they_are_not_allowed_in_this_part",
			err:   emailvalidator.ErrUnexpectedCharacter,
		},
	}

	for _, step := range steps {
		t.Run(step.label, func(t *testing.T) {
			_, err := emailvalidator.BuildResult(step.input)
			if step.err == nil {
				if err != nil {
					t.Logf("Test should not have failed but did: %v", err)
					t.Fail()
				}
			} else if err == nil {
				t.Log("Test should have failed but didn't")
				t.Fail()
			} else if !errors.Is(err, step.err) {
				t.Logf("Expected err to be %v but saw %v", step.err, err)
				t.Fail()
			}
		})
	}
}
