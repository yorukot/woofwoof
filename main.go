package main

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/spf13/cobra"
	"golang.org/x/text/unicode/norm"
)

var (
	// 8 cores × 8 tones = 64 tokens (fixed codebook)
	cores = []string{
		"汪",
		"嗚",
		"嗷",
		"汪汪",
		"嗚汪",
		"嗷汪",
		"汪嗚",
		"~汪",
	}
	tones = []string{
		"",   // 0
		".",  // 1
		"~",  // 2
		"～",  // 3 (fullwidth tilde)
		"…",  // 4 (ellipsis)
		"!",  // 5
		"！",  // 6 (fullwidth exclamation)
		"~.", // 7 (two-char tone, still no spaces)
	}

	codebook     []string
	reverseTable map[string]byte
)

func init() {
	codebook = make([]string, 0, 64)
	reverseTable = make(map[string]byte, 64)

	var id byte = 0
	for _, c := range cores {
		for _, t := range tones {
			token := c + t
			codebook = append(codebook, token)
			if _, exists := reverseTable[token]; exists {
				panic("duplicate token in codebook: " + token)
			}
			reverseTable[token] = id
			id++
		}
	}
	if len(codebook) != 64 {
		panic("codebook size is not 64")
	}
}

// Encode turns arbitrary UTF-8 text into dog-speech tokens.
func Encode(input string) (string, error) {
	// Normalize to NFC so visually-similar Unicode sequences become consistent.
	input = norm.NFC.String(input)

	// In Go, strings can contain invalid UTF-8; decide policy: reject invalid.
	if !utf8.ValidString(input) {
		return "", errors.New("input is not valid UTF-8")
	}

	payload := []byte(input)

	// Header: 4-byte length (big-endian)
	total := make([]byte, 4+len(payload))
	binary.BigEndian.PutUint32(total[:4], uint32(len(payload)))
	copy(total[4:], payload)

	// Convert bytes to 6-bit tokens
	var outTokens []string
	var bitBuf uint32
	var bitCount uint8

	emit6 := func(v byte) {
		outTokens = append(outTokens, codebook[v&0x3F])
	}

	for _, b := range total {
		bitBuf = (bitBuf << 8) | uint32(b)
		bitCount += 8
		for bitCount >= 6 {
			shift := bitCount - 6
			chunk := byte((bitBuf >> shift) & 0x3F)
			emit6(chunk)
			bitCount -= 6
			// keep remaining bits in bitBuf by masking
			if bitCount == 0 {
				bitBuf = 0
			} else {
				bitBuf = bitBuf & ((1 << bitCount) - 1)
			}
		}
	}

	// Pad remaining bits with zeros (safe because we have length header)
	if bitCount > 0 {
		chunk := byte((bitBuf << (6 - bitCount)) & 0x3F)
		emit6(chunk)
	}

	return strings.Join(outTokens, " "), nil
}

// Decode turns dog-speech tokens back into the original UTF-8 text.
func Decode(dogSpeech string) (string, error) {
	// Normalize NFC to reduce Unicode representation issues (esp. if copy/pasted).
	dogSpeech = norm.NFC.String(strings.TrimSpace(dogSpeech))
	if dogSpeech == "" {
		return "", errors.New("empty input")
	}

	parts := strings.Fields(dogSpeech) // splits on any whitespace; output format is still "space-separated"
	ids := make([]byte, 0, len(parts))
	for _, tok := range parts {
		id, ok := reverseTable[tok]
		if !ok {
			return "", fmt.Errorf("unknown token: %q", tok)
		}
		ids = append(ids, id)
	}

	// Convert 6-bit ids to bytes
	var bytesOut []byte
	var bitBuf uint32
	var bitCount uint8

	for _, id := range ids {
		bitBuf = (bitBuf << 6) | uint32(id&0x3F)
		bitCount += 6
		for bitCount >= 8 {
			shift := bitCount - 8
			b := byte((bitBuf >> shift) & 0xFF)
			bytesOut = append(bytesOut, b)
			bitCount -= 8
			if bitCount == 0 {
				bitBuf = 0
			} else {
				bitBuf = bitBuf & ((1 << bitCount) - 1)
			}
		}
	}

	// Need at least 4 bytes for length header
	if len(bytesOut) < 4 {
		return "", errors.New("decoded data too short (missing length header)")
	}
	n := binary.BigEndian.Uint32(bytesOut[:4])
	if int64(n) < 0 {
		return "", errors.New("invalid length header")
	}

	if len(bytesOut) < 4+int(n) {
		return "", fmt.Errorf("decoded data incomplete: need %d bytes payload, have %d", n, len(bytesOut)-4)
	}

	payload := bytesOut[4 : 4+int(n)]
	if !utf8.Valid(payload) {
		return "", errors.New("decoded payload is not valid UTF-8 (token stream may be corrupted)")
	}

	return string(payload), nil
}

func readAllStdin() (string, error) {
	in := bufio.NewReader(os.Stdin)
	b, err := in.ReadBytes(0)
	if err == nil {
		// unlikely to hit NUL; just in case
		return string(b[:len(b)-1]), nil
	}
	// If ReadBytes returns error, it usually includes partial data; fall back to ReadString loop
	// Simpler: read via scanner with big buffer
	sc := bufio.NewScanner(os.Stdin)
	// allow large inputs
	buf := make([]byte, 0, 1024*1024)
	sc.Buffer(buf, 10*1024*1024)
	var sb strings.Builder
	first := true
	for sc.Scan() {
		if !first {
			sb.WriteByte('\n')
		}
		first = false
		sb.WriteString(sc.Text())
	}
	if err := sc.Err(); err != nil {
		return "", err
	}
	return sb.String(), nil
}

func inputFromArgsOrStdin(args []string) (string, error) {
	if len(args) > 0 {
		return strings.Join(args, " "), nil
	}
	return readAllStdin()
}

func runMode(mode string, input string) (string, error) {
	switch strings.ToLower(mode) {
	case "encode", "enc":
		return Encode(input)
	case "decode", "dec":
		return Decode(input)
	default:
		return "", fmt.Errorf("unknown mode: %s", mode)
	}
}

func newRootCmd() *cobra.Command {
	var mode string

	rootCmd := &cobra.Command{
		Use:   "woofwoof [text]",
		Short: "Encode/decode text as dog speech",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			input, err := inputFromArgsOrStdin(args)
			if err != nil {
				return fmt.Errorf("read stdin error: %w", err)
			}
			out, err := runMode(mode, input)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), out)
			return nil
		},
	}
	rootCmd.Flags().StringVarP(&mode, "mode", "m", "encode", "encode or decode")

	encodeCmd := &cobra.Command{
		Use:   "encode [text]",
		Short: "Encode plain UTF-8 text to dog speech",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			input, err := inputFromArgsOrStdin(args)
			if err != nil {
				return fmt.Errorf("read stdin error: %w", err)
			}
			out, err := Encode(input)
			if err != nil {
				return fmt.Errorf("encode error: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), out)
			return nil
		},
	}

	decodeCmd := &cobra.Command{
		Use:   "decode [dog-speech]",
		Short: "Decode dog speech back to original UTF-8 text",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			input, err := inputFromArgsOrStdin(args)
			if err != nil {
				return fmt.Errorf("read stdin error: %w", err)
			}
			out, err := Decode(input)
			if err != nil {
				return fmt.Errorf("decode error: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), out)
			return nil
		},
	}

	rootCmd.AddCommand(encodeCmd, decodeCmd)
	return rootCmd
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
