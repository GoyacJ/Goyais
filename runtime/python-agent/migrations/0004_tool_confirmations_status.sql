-- v0.2.0 execution schema already stores confirmation status in execution_confirmations.
-- keep this migration slot as a no-op for migration ordering stability.
SELECT 1;
