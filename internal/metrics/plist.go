package metrics

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"io"
	"strconv"
	"strings"
)

// parsePlist decodes an XML property list into Go values: map[string]any,
// []any, string, int64, float64, bool and []byte.
func parsePlist(data []byte) (any, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false
	for {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		if se, ok := tok.(xml.StartElement); ok {
			if se.Name.Local == "plist" {
				continue
			}
			return plistValue(dec, se)
		}
	}
}

func plistValue(dec *xml.Decoder, se xml.StartElement) (any, error) {
	switch se.Name.Local {
	case "dict":
		m := map[string]any{}
		var key string
		for {
			tok, err := dec.Token()
			if err != nil {
				return nil, err
			}
			switch t := tok.(type) {
			case xml.StartElement:
				if t.Name.Local == "key" {
					s, err := plistText(dec)
					if err != nil {
						return nil, err
					}
					key = s
					continue
				}
				v, err := plistValue(dec, t)
				if err != nil {
					return nil, err
				}
				m[key] = v
			case xml.EndElement:
				return m, nil
			}
		}
	case "array":
		var a []any
		for {
			tok, err := dec.Token()
			if err != nil {
				return nil, err
			}
			switch t := tok.(type) {
			case xml.StartElement:
				v, err := plistValue(dec, t)
				if err != nil {
					return nil, err
				}
				a = append(a, v)
			case xml.EndElement:
				return a, nil
			}
		}
	case "true", "false":
		if err := dec.Skip(); err != nil {
			return nil, err
		}
		return se.Name.Local == "true", nil
	case "integer":
		s, err := plistText(dec)
		if err != nil {
			return nil, err
		}
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return i, nil
		}
		u, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return nil, err
		}
		return int64(u), nil // two's complement, as IOKit intends
	case "real":
		s, err := plistText(dec)
		if err != nil {
			return nil, err
		}
		return strconv.ParseFloat(s, 64)
	case "data":
		s, err := plistText(dec)
		if err != nil {
			return nil, err
		}
		return base64.StdEncoding.DecodeString(strings.Join(strings.Fields(s), ""))
	case "string", "date":
		return plistText(dec)
	}
	if err := dec.Skip(); err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	return nil, nil
}

func plistText(dec *xml.Decoder) (string, error) {
	var sb strings.Builder
	for {
		tok, err := dec.Token()
		if err != nil {
			return "", err
		}
		switch t := tok.(type) {
		case xml.CharData:
			sb.Write(t)
		case xml.EndElement:
			return strings.TrimSpace(sb.String()), nil
		}
	}
}

type plistDict map[string]any

func (d plistDict) int(k string) (int64, bool) {
	switch v := d[k].(type) {
	case int64:
		return v, true
	case float64:
		return int64(v), true
	}
	return 0, false
}

func (d plistDict) bool(k string) (bool, bool) {
	v, ok := d[k].(bool)
	return v, ok
}

func (d plistDict) str(k string) string {
	s, _ := d[k].(string)
	return s
}

func (d plistDict) dict(k string) plistDict {
	m, _ := d[k].(map[string]any)
	return m
}
