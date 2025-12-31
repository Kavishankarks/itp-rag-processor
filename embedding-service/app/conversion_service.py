import os
import shutil
from typing import Optional
from markitdown import MarkItDown
import pdfplumber


class ConversionService:
    def __init__(self):
        self.markitdown = MarkItDown()
        self.pdf_converter = os.getenv("PDF_CONVERTER", "pdfplumber").lower()

    def convert_file(self, file_path: str) -> str:
        """
        Convert a file to markdown using MarkItDown or pdfplumber for PDFs.
        
        Args:
            file_path: Path to the file to convert
            
        Returns:
            str: The converted markdown/text content
        """
        try:
            ext = os.path.splitext(file_path)[1].lower()
            
            if ext == ".pdf" and self.pdf_converter == "pdfplumber":
                text_content = []
                with pdfplumber.open(file_path) as pdf:
                    for page in pdf.pages:
                        text = page.extract_text()
                        if text:
                            text_content.append(text)
                return "\n\n".join(text_content)
            
            # Default to MarkItDown for other files or if configured
            result = self.markitdown.convert(file_path)
            return result.text_content
            
        except Exception as e:
            raise Exception(f"Failed to convert file: {str(e)}")


_conversion_service: Optional[ConversionService] = None


def get_conversion_service() -> ConversionService:
    global _conversion_service
    if _conversion_service is None:
        _conversion_service = ConversionService()
    return _conversion_service
