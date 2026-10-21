package main

import (
	"bytes"
	"os"
	"testing"
)

// The archive serves two scan formats inside its PDFs, and it also serves an
// HTML error page where a PDF should be. Both cases silently degraded whole
// volumes to 133x90 thumbnails; these tests pin the behaviour.

func TestJPEGFromPDFRejectsHTMLErrorPage(t *testing.T) {
	html := []byte("<HTML>\r\n<HEAD>\r\n\t<META HTTP-EQUIV=\"content-type\" CONTENT=\"text/html;charset=Shift-JIS\">\r\n<TITLE>Error Message</TITLE>\r\n</HEAD>\r\n<BODY>\r\n</BODY>\r\n</HTML>")
	if got := JPEGFromPDF(html); got != nil {
		t.Fatalf("HTML error page produced %d bytes of image data", len(got))
	}
	if got := JPEGFromPDF([]byte{}); got != nil {
		t.Fatalf("empty input produced %d bytes", len(got))
	}
	if got := JPEGFromPDF([]byte("%PDF-1.4\nno images here\n")); got != nil {
		t.Fatalf("PDF without images produced %d bytes", len(got))
	}
}

// buildDCTPDF wraps an arbitrary JPEG as a /DCTDecode stream, the layout the
// archive uses for its plain-JPEG scans.
func buildDCTPDF(jpegData []byte) []byte {
	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n4 0 obj\n<< /Filter /DCTDecode /Length ")
	b.WriteString(itoaAtLeast(len(jpegData)))
	b.WriteString(" >>\nstream\r\n")
	b.Write(jpegData)
	b.WriteString("\r\nendstream\nendobj\n")
	return b.Bytes()
}

// buildJPXPDF wraps a JPEG 2000 codestream as the archive's /JPXDecode pages.
func buildJPXPDF(jpx []byte) []byte {
	var b bytes.Buffer
	b.WriteString("%PDF-1.6\n5 0 obj\n<< /Filter /JPXDecode /Length ")
	b.WriteString(itoaAtLeast(len(jpx)))
	b.WriteString(" >>\nstream\r\n")
	b.Write(jpx)
	b.WriteString("\r\nendstream\nendobj\n")
	return b.Bytes()
}

func itoaAtLeast(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

func TestJPEGFromPDFExtractsDCTStream(t *testing.T) {
	jpegData := append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, bytes.Repeat([]byte{0x42}, 4096)...)
	jpegData = append(jpegData, 0xFF, 0xD9)

	got := JPEGFromPDF(buildDCTPDF(jpegData))
	if !bytes.Equal(got, jpegData) {
		t.Fatalf("DCT extraction mismatch: got %d bytes, want %d", len(got), len(jpegData))
	}
}

func TestJPEGFromPDFPicksLargestDCTStream(t *testing.T) {
	small := append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, bytes.Repeat([]byte{0x41}, 100)...)
	big := append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, bytes.Repeat([]byte{0x42}, 8000)...)

	pdf := bytes.Join([][]byte{
		[]byte("%PDF-1.4\n"),
		[]byte("<< /Filter /DCTDecode >>\nstream\r\n"), small, []byte("\r\nendstream\n"),
		[]byte("<< /Filter /DCTDecode >>\nstream\r\n"), big, []byte("\r\nendstream\n"),
	}, nil)

	got := JPEGFromPDF(pdf)
	if !bytes.Equal(got, big) {
		t.Fatalf("want the larger stream (%d bytes), got %d", len(big), len(got))
	}
}

// TestJPEGFromPDFDecodesJPX covers the JPEG 2000 scans. The fixture is a real
// codestream captured from the archive; EHONSHELF_JPX_FIXTURE may point at
// another one.
func TestJPEGFromPDFDecodesJPX(t *testing.T) {
	path := os.Getenv("EHONSHELF_JPX_FIXTURE")
	if path == "" {
		path = "testdata/jpx_sample.jpx"
	}
	jpx, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("no JPEG 2000 fixture at %s: %v", path, err)
	}

	got := JPEGFromPDF(buildJPXPDF(jpx))
	if len(got) < 1000 {
		t.Fatalf("JPX decode produced %d bytes, want a real JPEG", len(got))
	}
	if got[0] != 0xFF || got[1] != 0xD8 {
		t.Fatalf("JPX decode did not produce a JPEG: magic % x", got[:4])
	}
}
