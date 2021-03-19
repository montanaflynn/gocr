# PDF OCR

Convert PDFs to Mathpix Markdown, Microsoft Word or LaTeX:

## Install

```
curl -sf https://gobinaries.com/montanaflynn/gocr/cmd/gocr | sh
```

## Usage

Sign up for an OCR API key at https://accounts.mathpix.com then run `gocr`:

### Interactive

```
gocr
? What is your mathpix api key? ********************
? Which file should be used as an input? example.pdf
? Where would you like to save it? example.mmd
? Do you agree to the cost of $1.20 for 12 pages? Yes
✓ Saved example.mmd
```

### Non-interactive

```
export MATHPIX_OCR_API_KEY=...
gocr --agree example.pdf example.docx
✓ Saved example.docx
```

### CLI configuration, arguments and flags:

- You can set input and output paths as arguments `gocr input.pdf output.docx`
- You can set the API key with an environment variable `MATHPIX_OCR_API_KEY`
- You can set the API key with the flag `gocr --api-key ...`
- You can skip the cost agreement with the flag `gocr --agree`
- You can view all flags and arguments with `gocr --help`

