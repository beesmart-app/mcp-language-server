package protocol

import (
	"encoding/json"
	"strings"
)

// UnmarshalJSON decodifica Hover.contents nos formatos que o protocolo LSP
// permite historicamente: MarkupContent (recomendado pela spec atual), um
// MarkedString unico (string simples, ou {language, value}), ou um array de
// MarkedString. tsprotocol.go (gerado a partir da spec) so tipa Contents
// como MarkupContent, e o encoding/json padrao falha com "cannot unmarshal
// array into Go struct field" quando um language server manda o formato
// legado em array - foi observado no jdtls em respostas reais de hover.
func (h *Hover) UnmarshalJSON(data []byte) error {
	var raw struct {
		Contents json.RawMessage `json:"contents"`
		Range    Range           `json:"range,omitempty"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	h.Range = raw.Range
	h.Contents = parseHoverContents(raw.Contents)
	return nil
}

type markedString struct {
	Language string `json:"language"`
	Value    string `json:"value"`
}

func parseHoverContents(raw json.RawMessage) MarkupContent {
	if len(raw) == 0 {
		return MarkupContent{}
	}

	var markup MarkupContent
	if err := json.Unmarshal(raw, &markup); err == nil && markup.Value != "" {
		return markup
	}

	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return MarkupContent{Kind: PlainText, Value: s}
	}

	var single markedString
	if err := json.Unmarshal(raw, &single); err == nil && single.Value != "" {
		return MarkupContent{Kind: PlainText, Value: single.Value}
	}

	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err == nil {
		var parts []string
		for _, item := range items {
			var s string
			if err := json.Unmarshal(item, &s); err == nil && s != "" {
				parts = append(parts, s)
				continue
			}
			var m markedString
			if err := json.Unmarshal(item, &m); err == nil && m.Value != "" {
				parts = append(parts, m.Value)
			}
		}
		return MarkupContent{Kind: PlainText, Value: strings.Join(parts, "\n\n")}
	}

	return MarkupContent{}
}
