"""Tests for the deterministic embedding module (stdlib unittest, no deps)."""

import unittest

from embed import DEFAULT_MODEL, EMBED_DIM, embed_text, embed_texts


class EmbedTextTest(unittest.TestCase):
    """Verifies embed_text produces deterministic, normalised, fixed-dim vectors."""

    def test_dimension_is_fixed(self) -> None:
        vector = embed_text("refund_status")
        self.assertEqual(len(vector), EMBED_DIM)

    def test_is_deterministic(self) -> None:
        self.assertEqual(embed_text("refund_status"), embed_text("refund_status"))

    def test_is_l2_normalised(self) -> None:
        vector = embed_text("payment_status")
        norm = sum(component * component for component in vector) ** 0.5
        self.assertAlmostEqual(norm, 1.0, places=6)

    def test_empty_text_is_zero_vector(self) -> None:
        self.assertEqual(embed_text(""), [0.0] * EMBED_DIM)

    def test_similar_names_score_higher_than_unrelated(self) -> None:
        a = embed_text("refund_status")
        b = embed_text("refundStatus")
        c = embed_text("invoice_total")

        def cosine(x: list[float], y: list[float]) -> float:
            return sum(xi * yi for xi, yi in zip(x, y))

        self.assertGreater(cosine(a, b), cosine(a, c))


class EmbedTextsTest(unittest.TestCase):
    """Verifies embed_texts preserves order and the model label is exposed."""

    def test_preserves_order(self) -> None:
        vectors = embed_texts(["refund_status", "invoice_total"])
        self.assertEqual(len(vectors), 2)
        self.assertEqual(vectors[0], embed_text("refund_status"))
        self.assertEqual(vectors[1], embed_text("invoice_total"))

    def test_model_label_is_stable(self) -> None:
        self.assertEqual(DEFAULT_MODEL, "contextos-hashing-v1")


if __name__ == "__main__":
    unittest.main()
