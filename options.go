package slackdown

import (
	"io"
	"sync"
)

type Option func(*Renderer)

func WithMentionTranslation() Option {
	return func(r *Renderer) {
		r.processMentions = processMentions(r)
	}
}

func processMentions(r *Renderer) func(io.Writer, []byte) {
	mRxp := mentionRxp
	tmpl := []byte(`<!$mention>`)
	pool := &sync.Pool{
		New: func() interface{} {
			buf := make([]byte, 0, 128)
			return &buf
		},
	}

	processMentionsClosure := func(w io.Writer, b []byte) {
		if !mRxp.Match(b) {
			r.esc(w, b)
			return
		}

		bufp := pool.Get().(*[]byte)
		defer pool.Put(bufp)

		buf := *bufp
		buf = buf[:0]
		offset := 0

		for _, submatches := range mRxp.FindAllSubmatchIndex(b, -1) {
			buf = mRxp.Expand(buf, tmpl, b, submatches)

			r.esc(w, b[offset:submatches[2]-1])
			r.out(w, buf)

			buf = buf[:0]
			offset = submatches[3]
		}
		r.esc(w, b[offset:])
	}

	return processMentionsClosure
}
