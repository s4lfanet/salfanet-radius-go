// Package qris provides utilities for QRIS (Indonesian QR payment) processing.
// It converts a static QRIS string (from your bank) to a dynamic one with a
// specific transaction amount, following the EMVCo QR Code TLV standard.
//
// Tag reference:
//
//	00 = Payload Format Indicator
//	01 = Point of Initiation Method (11=static, 12=dynamic)
//	26-51 = Merchant Account Information
//	52 = Merchant Category Code
//	53 = Transaction Currency (360=IDR)
//	54 = Transaction Amount
//	58 = Country Code (ID)
//	59 = Merchant Name
//	60 = Merchant City
//	63 = CRC (CRC-16/CCITT-FALSE)
package qris

import (
	"fmt"
	"strconv"
)

// tlvField represents a single TLV (Tag-Length-Value) field.
type tlvField struct {
	tag   string
	value string
}

// parseTLV parses a QRIS string into TLV fields.
func parseTLV(data string) []tlvField {
	var fields []tlvField
	i := 0
	for i+4 <= len(data) {
		tag := data[i : i+2]
		length, err := strconv.Atoi(data[i+2 : i+4])
		if err != nil || i+4+length > len(data) {
			break
		}
		value := data[i+4 : i+4+length]
		fields = append(fields, tlvField{tag: tag, value: value})
		i += 4 + length
	}
	return fields
}

// buildTLV serialises TLV fields back into a QRIS string.
func buildTLV(fields []tlvField) string {
	result := ""
	for _, f := range fields {
		result += f.tag + fmt.Sprintf("%02d", len(f.value)) + f.value
	}
	return result
}

// crc16ccitt calculates CRC-16/CCITT-FALSE over the given string.
func crc16ccitt(data string) string {
	crc := uint16(0xFFFF)
	for i := 0; i < len(data); i++ {
		crc ^= uint16(data[i]) << 8
		for j := 0; j < 8; j++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return fmt.Sprintf("%04X", crc)
}

// StaticToDynamic converts a static QRIS string to a dynamic one with the
// given IDR amount embedded. Returns an error if the input is invalid.
func StaticToDynamic(staticQris string, amount int) (string, error) {
	if staticQris == "" || amount <= 0 {
		return "", fmt.Errorf("invalid QRIS string or amount")
	}

	fields := parseTLV(staticQris)
	if len(fields) == 0 {
		return "", fmt.Errorf("failed to parse QRIS TLV structure")
	}

	// Drop CRC field (tag 63) — will recalculate
	var withoutCRC []tlvField
	for _, f := range fields {
		if f.tag != "63" {
			withoutCRC = append(withoutCRC, f)
		}
	}

	// Update / insert required fields
	var updated []tlvField
	hasTag01, hasTag54 := false, false

	for _, f := range withoutCRC {
		switch f.tag {
		case "01":
			updated = append(updated, tlvField{tag: "01", value: "12"}) // static → dynamic
			hasTag01 = true
		case "54":
			updated = append(updated, tlvField{tag: "54", value: strconv.Itoa(amount)})
			hasTag54 = true
		default:
			updated = append(updated, f)
		}
	}

	if !hasTag01 {
		// Insert after tag 00
		var tmp []tlvField
		for i, f := range updated {
			tmp = append(tmp, f)
			if f.tag == "00" {
				tmp = append(tmp, tlvField{tag: "01", value: "12"})
				tmp = append(tmp, updated[i+1:]...)
				break
			}
		}
		updated = tmp
	}

	if !hasTag54 {
		// Insert after tag 53 (currency), or before tag 58 (country), or at end
		insertAt := len(updated)
		for i, f := range updated {
			if f.tag == "53" {
				insertAt = i + 1
				break
			}
			if f.tag == "58" {
				insertAt = i
				break
			}
		}
		tail := make([]tlvField, len(updated)-insertAt)
		copy(tail, updated[insertAt:])
		updated = append(updated[:insertAt], tlvField{tag: "54", value: strconv.Itoa(amount)})
		updated = append(updated, tail...)
	}

	// Build string and append CRC
	tlvString := buildTLV(updated)
	withCRCHeader := tlvString + "6304"
	crc := crc16ccitt(withCRCHeader)
	return withCRCHeader + crc, nil
}

// ValidateQris checks the CRC of a QRIS string.
func ValidateQris(qris string) bool {
	if len(qris) < 10 {
		return false
	}
	data := qris[:len(qris)-4]
	existing := qris[len(qris)-4:]
	for i, c := range existing {
		if c >= 'a' && c <= 'f' {
			existing = existing[:i] + string(c-32) + existing[i+1:]
		}
	}
	return crc16ccitt(data) == existing
}
