package slackdown

import (
	"bytes"
	"fmt"
	"io"
	"strconv"

	bf "github.com/russross/blackfriday/v2"
)

var (
	strongTag        = []byte("*")
	strikethroughTag = []byte("~")
	itemTag          = []byte("-")
	codeTag          = []byte("`")
	codeBlockTag     = []byte("```")
	linkTag          = []byte("<")
	linkCloseTag     = []byte(">")
	pipeSign         = []byte("|")
	italicTag        = []byte("_")
	// pseudo table delim
	tableRowTag    = []byte("• ")
	tableColumnTag = []byte(" *|*")

	nlBytes    = []byte{'\n'}
	spaceBytes = []byte{' '}

	escapes = [256][]byte{
		'&': []byte(`&amp;`),
		'<': []byte(`&lt;`),
		'>': []byte(`&gt;`),
	}
)

// Renderer is the rendering interface for slack output.
type Renderer struct {
	lastOutputLen     int
	itemLevel         int
	itemListDepth     map[int]int
	tableRowBeginning bool
	err               error
}

func NewRenderer() *Renderer {
	return &Renderer{
		itemListDepth: make(map[int]int),
	}
}

// Err will return a non-nil error if Render methods failed, usually
// from an io.Writer error. It should be called after a Render call
// but before a call to Reset.
func (r *Renderer) Err() error { return r.err }

// Reset optionally resets the internal state of the Renderer for reuse.
func (r *Renderer) Reset() {
	r.lastOutputLen = 0
	r.itemLevel = 0
	r.err = nil
	r.tableRowBeginning = false

	for k := range r.itemListDepth {
		delete(r.itemListDepth, k)
	}
}

func (r *Renderer) esc(w io.Writer, text []byte) {
	if r.err != nil {
		return
	}

	var start, end, n int
	for ; end < len(text); end++ {
		escSeq := escapes[text[end]]
		if escSeq == nil {
			continue
		}

		m, e1 := w.Write(text[start:end])
		n, r.err = w.Write(escSeq)
		start = end + 1
		r.lastOutputLen += n + m

		if e1 != nil {
			r.err = e1
		}

		if r.err != nil {
			return
		}
	}

	if start < len(text) && end <= len(text) {
		n, r.err = w.Write(text[start:end])
		r.lastOutputLen += n
	}
}

func (r *Renderer) out(w io.Writer, text []byte) {
	if r.err != nil {
		return
	}

	n, err := w.Write(text)
	r.lastOutputLen = n

	r.err = err
}

func (r *Renderer) cr(w io.Writer) {
	if r.lastOutputLen > 0 {
		r.out(w, nlBytes)
	}
}

// RenderNode parses a single node of a syntax tree.
func (r *Renderer) RenderNode(w io.Writer, node *bf.Node, entering bool) bf.WalkStatus {

	switch node.Type {
	case bf.Text:
		r.esc(w, node.Literal)
	case bf.Softbreak:

	case bf.Hardbreak:

	case bf.BlockQuote:

	case bf.CodeBlock:
		r.out(w, codeBlockTag)
		r.esc(w, node.Literal)
		r.out(w, codeBlockTag)
		r.cr(w)
		r.cr(w)

	case bf.Code:
		r.out(w, codeTag)
		r.esc(w, node.Literal)
		r.out(w, codeTag)

	case bf.Emph:

	case bf.Heading:
		if entering {
			r.out(w, strongTag)
		} else {
			r.out(w, strongTag)
			r.cr(w)
		}
	case bf.Image:

	case bf.Item:
		if entering {
			r.out(w, spaceBytes)
			for i := 1; i < r.itemLevel; i++ {
				r.out(w, spaceBytes)
				r.out(w, spaceBytes)
				r.out(w, spaceBytes)
			}
			if node.ListFlags&bf.ListTypeOrdered != 0 {
				r.out(w, append([]byte(strconv.Itoa(r.itemListDepth[r.itemLevel])), node.ListData.Delimiter))
				r.itemListDepth[r.itemLevel]++
			} else {
				r.out(w, itemTag)
			}
			r.out(w, spaceBytes)
		}

	case bf.Link:
		if entering {
			r.out(w, linkTag)
			if dest := node.LinkData.Destination; dest != nil {
				r.out(w, dest)
				r.out(w, pipeSign)
			}
		} else {
			r.out(w, linkCloseTag)
		}

	case bf.HorizontalRule:

	case bf.List:
		if entering {
			r.itemLevel++
			if node.ListFlags&bf.ListTypeOrdered != 0 {
				r.itemListDepth[r.itemLevel] = 1
			}
		} else {
			if node.ListFlags&bf.ListTypeOrdered != 0 {
				delete(r.itemListDepth, r.itemLevel)
			}
			r.itemLevel--
			if r.itemLevel == 0 {
				r.cr(w)
			}
		}

	case bf.Document:

	case bf.Paragraph:
		if !entering {
			if node.Parent.Type != bf.Item {
				r.cr(w)
			}
			r.cr(w)
		}

	case bf.Strong:
		r.out(w, strongTag)

	case bf.Del:
		r.out(w, strikethroughTag)

	case bf.HTMLBlock:

	case bf.HTMLSpan:

	case bf.Table:

	case bf.TableCell:
		switch {
		case node.IsHeader && entering:
			if !r.tableRowBeginning {
				r.out(w, spaceBytes)
			}

			r.out(w, italicTag)
			r.out(w, strongTag)
			r.tableRowBeginning = false

		case node.IsHeader && !entering:
			r.out(w, strongTag)
			r.out(w, italicTag)
			r.out(w, tableColumnTag)

		case entering && r.tableRowBeginning:
			r.out(w, tableRowTag)
			r.tableRowBeginning = false

		case !entering && !r.tableRowBeginning:
			r.out(w, tableColumnTag)

		default:
			// 'entering' is true and the others are false.
			r.out(w, spaceBytes)
		}

	case bf.TableHead:

	case bf.TableBody:

	case bf.TableRow:
		if entering {
			r.cr(w)
		}
		r.tableRowBeginning = entering

	default:
		if r.err == nil {
			r.err = fmt.Errorf("unknown node type: %s", node.Type)
		}
	}

	if r.err != nil {
		return bf.Terminate
	}

	return bf.GoToNext
}

// Render prints out the whole document from the ast.
// A call to Err will return non-nil if rendering stopped early,
// usually due to an internal writting error.
func (r *Renderer) Render(ast *bf.Node) []byte {
	buf := new(bytes.Buffer)
	r.RenderOut(buf, ast)

	return buf.Bytes()
}

// RenderOut renders the ast to the provided io.Writer. An error
// is returned if rendering stopped early, usually due to an io.Writer error.
// The error returned from this method is the same as a call to Err.
func (r *Renderer) RenderOut(w io.Writer, ast *bf.Node) error {
	ast.Walk(func(node *bf.Node, entering bool) bf.WalkStatus {
		return r.RenderNode(w, node, entering)
	})

	return r.Err()
}

// RenderHeader writes document header (unused).
func (r *Renderer) RenderHeader(w io.Writer, ast *bf.Node) {
}

// RenderFooter writes document footer (unused).
func (r *Renderer) RenderFooter(w io.Writer, ast *bf.Node) {
}

// Run prints out the confluence document.
func Run(input []byte, opts ...bf.Option) []byte {
	r := NewRenderer()
	optList := []bf.Option{bf.WithRenderer(r), bf.WithExtensions(bf.CommonExtensions)}
	optList = append(optList, opts...)
	parser := bf.New(optList...)
	ast := parser.Parse([]byte(input))
	return r.Render(ast)
}
