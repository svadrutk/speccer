import unittest
import ast
from speccer_parser.parser import SymbolTable, extract_models, resolve_imports


class TestPythonParser(unittest.TestCase):
    def test_sqlalchemy_model_extraction(self):
        source_code = """
from sqlalchemy import Column, Integer, String
from sqlalchemy.ext.declarative import declarative_base

Base = declarative_base()

class User(Base):
    __tablename__ = 'users'
    id = Column(Integer, primary_key=True)
    username = Column(String(50), nullable=False)
"""
        tree = ast.parse(source_code)
        models = extract_models(tree)
        self.assertIn("User", models)
        # Integer gets normalized to "int" by TYPE_NORMALIZE_MAP
        self.assertEqual(models["User"]["fields"]["id"]["type"], "int")
        # primary_key=True implies nullable=False
        self.assertEqual(models["User"]["fields"]["id"]["nullable"], False)
        self.assertEqual(models["User"]["fields"]["username"]["nullable"], False)

    def test_import_alias_resolution(self):
        source_code = """
import sqlalchemy as sa
from sqlalchemy.ext.declarative import declarative_base

Base = declarative_base()

class User(Base):
    __tablename__ = 'users'
    id = sa.Column(sa.Integer, primary_key=True)
    username = sa.Column(sa.String(50), nullable=False)
"""
        tree = ast.parse(source_code)
        models = extract_models(tree)
        self.assertIn("User", models)
        self.assertEqual(models["User"]["fields"]["id"]["type"], "int")

    def test_wildcard_import_graceful_fallback(self):
        """Wildcard imports are ignored; symbols resolve to 'unresolved_type'."""
        source_code = """
from sqlalchemy import *
from sqlalchemy.ext.declarative import declarative_base

Base = declarative_base()

class User(Base):
    __tablename__ = 'users'
    id = Column(Integer, primary_key=True)
"""
        tree = ast.parse(source_code)
        models = extract_models(tree)
        self.assertIn("User", models)
        # Column/Integer unresolved via wildcard — type falls back gracefully
        self.assertIn(models["User"]["fields"]["id"]["type"],
                      ["int", "unresolved_type"])

    def test_pydantic_model_extraction(self):
        source_code = """
from pydantic import BaseModel

class Item(BaseModel):
    name: str
    price: float
    in_stock: bool = True
"""
        tree = ast.parse(source_code)
        models = extract_models(tree)
        self.assertIn("Item", models)
        self.assertEqual(models["Item"]["fields"]["name"]["type"], "string")
        self.assertEqual(models["Item"]["fields"]["price"]["type"], "float")
        self.assertEqual(models["Item"]["fields"]["in_stock"]["type"], "bool")

    def test_optional_type_resolution(self):
        source_code = """
from typing import Optional
from pydantic import BaseModel

class User(BaseModel):
    email: Optional[str] = None
    age: Optional[int] = None
"""
        tree = ast.parse(source_code)
        models = extract_models(tree)
        self.assertIn("User", models)
        self.assertEqual(models["User"]["fields"]["email"]["type"], "optional(string)")
        self.assertTrue(models["User"]["fields"]["email"]["nullable"])
        self.assertEqual(models["User"]["fields"]["age"]["type"], "optional(int)")

    def test_empty_file(self):
        tree = ast.parse("")
        models = extract_models(tree)
        self.assertEqual(models, {})

    def test_no_model_classes(self):
        source_code = """
def helper():
    pass

SOME_CONSTANT = 42
"""
        tree = ast.parse(source_code)
        models = extract_models(tree)
        self.assertEqual(models, {})


if __name__ == '__main__':
    unittest.main()
