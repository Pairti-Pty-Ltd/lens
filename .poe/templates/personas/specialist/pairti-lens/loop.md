# Diagnose a Lens regression

Keep the same document, writer action, and expected result across attempts. Locate the first divergence in event handling, buffer state, Markdown parsing, canvas painting, or report generation.

Run the smallest Go test that exercises that boundary, then repeat the original interaction. For export failures, inspect the generated artifact. For visual failures, record the drawn or window result. Report the reproduction and remaining discrepancy, if any.
