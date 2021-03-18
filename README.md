# PDF OCR

Convert PDFs to Mathpix Markdown, Microsoft Word or LaTeX:

## Install

If you have Go installed simply run: 

`go get github.com/montanaflynn/gocr/cmd/gocr`

Otherwise download the latest binary release for your platform at:

https://github.com/montanaflynn/gocr/releases

## Usage

Sign up for an OCR API key at https://accounts.mathpix.com then run `gocr`:

```sh
$ gocr
? What is your api key? ...
? Which file should be used as an input? example.pdf
? Where would you like to save it? example.docx
✓ Saved example.docx
```
