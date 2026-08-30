package playback

import (
	"net/http"
)

func writeRedirect(w http.ResponseWriter, r *http.Request, res Resolved, intent Intent) {
	writeDynamicRedirectHeaders(w.Header())
	http.Redirect(w, r, res.Link.URL, http.StatusFound)
}

// 禁止缓存 302，避免过期直链
func writeDynamicRedirectHeaders(header http.Header) {
	header.Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	header.Set("Pragma", "no-cache")
	header.Set("Expires", "0")
	header.Set("Referrer-Policy", "no-referrer")
}
