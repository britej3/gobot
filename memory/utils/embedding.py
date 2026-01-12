"""
Embedding Model Factory
Routes to Ollama or SentenceTransformers based on config
"""
from typing import List
import numpy as np
import config


def get_embedding_model(model_name: str = None, use_optimization: bool = True):
    """
    Factory function that returns appropriate embedding model based on model name
    
    - "ollama:*" -> OllamaEmbeddingModel (local Ollama)
    - Otherwise -> SentenceTransformers (HuggingFace)
    """
    model_name = model_name or config.EMBEDDING_MODEL
    
    if model_name.startswith("ollama:"):
        from utils.ollama_embedding import OllamaEmbeddingModel
        return OllamaEmbeddingModel(model_name)
    else:
        from utils.embedding_original import EmbeddingModel as SentenceTransformerEmbedding
        return SentenceTransformerEmbedding(model_name, use_optimization)


class EmbeddingModel:
    """
    Embedding model wrapper - auto-selects backend based on config
    """
    def __init__(self, model_name: str = None, use_optimization: bool = True):
        self._model = get_embedding_model(model_name, use_optimization)
        
        # Expose attributes from underlying model
        self.model_name = self._model.model_name
        self.model_type = self._model.model_type
        self.dimension = self._model.dimension
        self.supports_query_prompt = getattr(self._model, 'supports_query_prompt', False)
    
    def encode(self, texts: List[str], is_query: bool = False) -> np.ndarray:
        """Encode texts to vectors"""
        return self._model.encode(texts, is_query=is_query)
    
    def encode_single(self, text: str, is_query: bool = False) -> np.ndarray:
        """Encode single text"""
        return self._model.encode_single(text, is_query=is_query)
    
    def encode_query(self, queries: List[str]) -> np.ndarray:
        """Encode queries"""
        return self._model.encode_query(queries)
    
    def encode_documents(self, documents: List[str]) -> np.ndarray:
        """Encode documents"""
        return self._model.encode_documents(documents)
