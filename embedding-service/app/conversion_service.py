import os
import shutil
from typing import Optional
from markitdown import MarkItDown


class ConversionService:
    def __init__(self):
        self.markitdown = MarkItDown()

    def convert_file(self, file_path: str) -> str:
        """
        Convert a file to markdown using MarkItDown.
        
        Args:
            file_path: Path to the file to convert
            
        Returns:
            str: The converted markdown content
        """
        try:
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
