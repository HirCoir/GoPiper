package main

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"unicode"
)

// Filter code blocks from text
func filterCodeBlocks(text string) string {
	re := regexp.MustCompile("(?s)```[^`\\n]*\\n.*?```")
	return re.ReplaceAllString(text, "")
}

// Process line breaks
func processLineBreaks(text string) string {
	log.Printf("[LINE_BREAKS] Original text: \"%s\"", truncateString(text, 200))

	processedText := text

	// Handle paragraph breaks (double line breaks)
	processedText = regexp.MustCompile(`\n\s*\n`).ReplaceAllString(processedText, ". ")

	// Handle single line breaks more carefully
	processedText = regexp.MustCompile(`([.!?¿¡…])\s*\n`).ReplaceAllString(processedText, "$1 ")
	processedText = regexp.MustCompile(`([^.!?¿¡…])\s*\n\s*([A-ZÁÉÍÓÚÑÜ])`).ReplaceAllString(processedText, "$1. $2")
	processedText = strings.ReplaceAll(processedText, "\n", " ")

	// Clean up spacing
	processedText = regexp.MustCompile(`\s+`).ReplaceAllString(processedText, " ")
	processedText = strings.TrimSpace(processedText)

	// Handle special cases for better speech flow
	processedText = regexp.MustCompile(`([a-zA-Z])\s*:\s*`).ReplaceAllString(processedText, "$1: ")

	// Clean up multiple periods
	processedText = regexp.MustCompile(`\.{4,}`).ReplaceAllString(processedText, "...")
	// Replace exactly 2 dots with 1 (but preserve 3+ dots which are ellipsis)
	processedText = regexp.MustCompile(`\.{2}`).ReplaceAllStringFunc(processedText, func(match string) string {
		// Check if it's part of ellipsis by looking at context
		return "."
	})
	// Restore ellipsis if broken
	processedText = regexp.MustCompile(`\.{3,}`).ReplaceAllString(processedText, "...")

	log.Printf("[LINE_BREAKS] Final processed text: \"%s\"", truncateString(processedText, 200))

	return processedText
}

// Apply text replacements
func applyReplacements(text string, replacements [][]string) string {
	if text == "" || len(replacements) == 0 {
		return text
	}

	log.Printf("[REPLACEMENTS] Starting text: '%s'", truncateString(text, 100))
	processedText := text

	for _, replacement := range replacements {
		if len(replacement) < 2 {
			continue
		}

		find := replacement[0]
		replace := replacement[1]

		if find == "" {
			continue
		}

		// Escape special regex characters
		escapedFind := regexp.QuoteMeta(find)

		// Count occurrences before
		beforeRe := regexp.MustCompile(escapedFind)
		beforeCount := len(beforeRe.FindAllString(processedText, -1))

		// Apply replacement based on pattern type
		if strings.HasSuffix(find, ".") {
			// Abbreviations ending with period
			pattern := regexp.MustCompile(`(?i)\b` + escapedFind)
			processedText = pattern.ReplaceAllString(processedText, replace)
		} else if strings.Contains(find, " ") {
			// Multi-word phrases
			pattern := regexp.MustCompile(`(?i)\b` + escapedFind + `\b`)
			processedText = pattern.ReplaceAllString(processedText, replace)
		} else if regexp.MustCompile(`^\d+$`).MatchString(find) {
			// Numbers - replace only standalone numbers, not part of larger numbers
			pattern := regexp.MustCompile(`(?i)\b` + escapedFind + `\b`)
			processedText = pattern.ReplaceAllString(processedText, replace)
		} else {
			// Standard word boundaries
			pattern := regexp.MustCompile(`(?i)\b` + escapedFind + `\b`)
			processedText = pattern.ReplaceAllString(processedText, replace)
		}

		// Count occurrences after
		afterCount := len(beforeRe.FindAllString(processedText, -1))
		replacementsMade := beforeCount - afterCount

		if replacementsMade > 0 {
			log.Printf("[REPLACEMENTS] '%s' → '%s' (%d replacements)", find, replace, replacementsMade)
		}
	}

	if processedText != text {
		log.Printf("[REPLACEMENTS] Final text: '%s'", truncateString(processedText, 100))
	} else {
		log.Println("[REPLACEMENTS] No changes made to text")
	}

	return processedText
}

// Normalize text for TTS
func normalizeTextForTTS(text string) string {
	log.Printf("[NORMALIZE] Starting normalization: \"%s\"", truncateString(text, 100))

	normalized := text

	// Handle line breaks
	normalized = regexp.MustCompile(`\n\s*\n`).ReplaceAllString(normalized, ". ")
	normalized = strings.ReplaceAll(normalized, "\n", " ")

	// Normalize quotes and dashes
	normalized = strings.ReplaceAll(normalized, "\u201c", "\"") // "
	normalized = strings.ReplaceAll(normalized, "\u201d", "\"") // "
	normalized = strings.ReplaceAll(normalized, "\u2018", "\"") // '
	normalized = strings.ReplaceAll(normalized, "\u2019", "\"") // '
	normalized = strings.ReplaceAll(normalized, "\u2013", "-")  // –
	normalized = strings.ReplaceAll(normalized, "\u2014", "-")  // —
	normalized = strings.ReplaceAll(normalized, "\u2026", "...") // …

	// Fix malformed punctuation combinations
	normalized = strings.ReplaceAll(normalized, "¿¡", "¿")
	normalized = strings.ReplaceAll(normalized, "¡¿", "¡")
	normalized = strings.ReplaceAll(normalized, "?!", "?")
	normalized = strings.ReplaceAll(normalized, "!?", "!")

	// Remove duplicate punctuation marks
	normalized = regexp.MustCompile(`¿¿+`).ReplaceAllString(normalized, "¿")
	normalized = regexp.MustCompile(`¡¡+`).ReplaceAllString(normalized, "¡")
	normalized = regexp.MustCompile(`\?\?+`).ReplaceAllString(normalized, "?")
	normalized = regexp.MustCompile(`!!+`).ReplaceAllString(normalized, "!")

	// Ensure proper question format
	normalized = regexp.MustCompile(`¿([^?]*?)\?`).ReplaceAllStringFunc(normalized, func(match string) string {
		content := strings.TrimPrefix(strings.TrimSuffix(match, "?"), "¿")
		return "¿" + strings.TrimSpace(content) + "?"
	})

	// Ensure proper exclamation format
	normalized = regexp.MustCompile(`¡([^!]*?)!`).ReplaceAllStringFunc(normalized, func(match string) string {
		content := strings.TrimPrefix(strings.TrimSuffix(match, "!"), "¡")
		return "¡" + strings.TrimSpace(content) + "!"
	})

	// Fix incomplete patterns - questions starting with ¿ but ending with .
	normalized = regexp.MustCompile(`¿\s*([^?]*?)\.`).ReplaceAllString(normalized, "¿$1?")
	// Fix incomplete patterns - exclamations starting with ¡ but ending with .
	normalized = regexp.MustCompile(`¡\s*([^!]*?)\.`).ReplaceAllString(normalized, "¡$1!")

	// Fix sentences ending with colon
	normalized = regexp.MustCompile(`:\s*$`).ReplaceAllString(normalized, ".")
	// Replace colon followed by uppercase letter with period and space
	normalized = regexp.MustCompile(`:\s*([A-ZÁÉÍÓÚÑÜ])`).ReplaceAllString(normalized, ". $1")

	// Clean up spacing
	normalized = regexp.MustCompile(`\s+([.!?¿¡,;:])`).ReplaceAllString(normalized, "$1")
	normalized = regexp.MustCompile(`([.!?])\s*([¿¡])`).ReplaceAllString(normalized, "$1 $2")

	// Ensure proper spacing after punctuation
	normalized = regexp.MustCompile(`([.!?])\s*([A-ZÁÉÍÓÚÑÜ])`).ReplaceAllString(normalized, "$1 $2")
	normalized = regexp.MustCompile(`([,:;])\s*([A-ZÁÉÍÓÚÑÜ])`).ReplaceAllString(normalized, "$1 $2")

	// Clean up multiple periods
	normalized = regexp.MustCompile(`\.{4,}`).ReplaceAllString(normalized, "...")
	// Replace exactly 2 dots with 1 (preserving ellipsis)
	for strings.Contains(normalized, "..") && !strings.Contains(normalized, "...") {
		normalized = strings.ReplaceAll(normalized, "..", ".")
	}

	// Remove duplicate punctuation
	normalized = regexp.MustCompile(`([!?]){2,}`).ReplaceAllString(normalized, "$1")

	// Normalize whitespace
	normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, " ")
	normalized = strings.TrimSpace(normalized)

	log.Printf("[NORMALIZE] Final result: \"%s\"", truncateString(normalized, 100))
	return normalized
}

// Split text into sentences
func splitSentences(text string) []string {
	if strings.TrimSpace(text) == "" {
		return []string{}
	}

	log.Printf("[SPLIT] Original text: \"%s\"", text)

	// Normalize text first
	normalizedText := normalizeTextForTTS(text)

	// Common abbreviations - ONLY real abbreviations that end with period
	abbreviations := []string{
		// Spanish titles
		"Sr.", "Sra.", "Srta.", "Dr.", "Dra.", "Prof.", "Profa.", 
		"Lic.", "Licda.", "Ing.", "Inga.", "Arq.", "Arqa.", 
		"Mtro.", "Mtra.",
		// Common abbreviations
		"etc.", "vs.", "p.ej.",
		// English abbreviations
		"Mr.", "Mrs.", "Ms.", "Inc.", "Ltd.", "Corp.", "Co.",
		"e.g.", "i.e.", "cf.", "vol.", "cap.", "art.", 
		"núm.", "pág.", "ed.", "op.cit.",
	}

	// Protect abbreviations by replacing them with placeholders
	protectedText := normalizedText
	protectionMap := make(map[string]string)
	
	for i, abbrev := range abbreviations {
		placeholder := fmt.Sprintf("__ABBREV_%d__", i)
		// Simple string replacement - no regex needed
		protectedText = strings.ReplaceAll(protectedText, abbrev, placeholder)
		protectionMap[placeholder] = abbrev
	}

	// Split sentences more intelligently
	// Look for: period/question/exclamation + space + (uppercase letter OR start of new paragraph)
	// But NOT if it's a single letter followed by period (like "S. i" which should be "Si")
	
	sentences := []string{}
	currentSentence := ""
	runes := []rune(protectedText)
	
	for i := 0; i < len(runes); i++ {
		currentSentence += string(runes[i])
		
		// Check if this is a sentence boundary
		if (runes[i] == '.' || runes[i] == '!' || runes[i] == '?') {
			// Look ahead to see what comes next
			if i+1 < len(runes) {
				// Skip whitespace to find next meaningful character
				nextMeaningfulIdx := i + 1
				for nextMeaningfulIdx < len(runes) && unicode.IsSpace(runes[nextMeaningfulIdx]) {
					nextMeaningfulIdx++
				}
				
				if nextMeaningfulIdx < len(runes) {
					nextMeaningful := runes[nextMeaningfulIdx]
					
					// This is a sentence boundary if:
					// 1. Next character is uppercase AND
					// 2. Current sentence has at least 10 characters (avoid splitting "S. i" -> "S." + "i...")
					// 3. OR next character is opening punctuation (¿¡)
					if (unicode.IsUpper(nextMeaningful) && len(strings.TrimSpace(currentSentence)) > 10) ||
						nextMeaningful == '¿' || nextMeaningful == '¡' {
						
						// This is a real sentence boundary
						sentence := strings.TrimSpace(currentSentence)
						if len(sentence) > 3 {
							sentences = append(sentences, sentence)
							log.Printf("[SPLIT] Extracted sentence: \"%s\"", truncateString(sentence, 80))
						}
						currentSentence = ""
					}
				}
			} else {
				// End of text
				sentence := strings.TrimSpace(currentSentence)
				if len(sentence) > 3 {
					sentences = append(sentences, sentence)
					log.Printf("[SPLIT] Extracted final sentence: \"%s\"", truncateString(sentence, 80))
				}
				currentSentence = ""
			}
		}
	}
	
	// Add any remaining text
	if len(strings.TrimSpace(currentSentence)) > 3 {
		sentences = append(sentences, strings.TrimSpace(currentSentence))
		log.Printf("[SPLIT] Extracted remaining text: \"%s\"", truncateString(currentSentence, 80))
	}

	// Process and clean sentences
	processedSentences := []string{}
	for _, sentence := range sentences {
		// Restore abbreviations
		for placeholder, original := range protectionMap {
			sentence = strings.ReplaceAll(sentence, placeholder, original)
		}

		// Enhance sentence
		sentence = enhanceSentenceForTTS(sentence)

		if len(sentence) > 3 {
			// Split long sentences
			if len(sentence) > 400 {
				chunks := splitLongSentence(sentence)
				processedSentences = append(processedSentences, chunks...)
			} else {
				processedSentences = append(processedSentences, sentence)
			}
		}
	}

	// Merge short fragments
	finalSentences := mergeShortFragments(processedSentences)

	if len(finalSentences) > 0 {
		log.Printf("[SPLIT] Text divided into %d segments:", len(finalSentences))
		for i, sentence := range finalSentences {
			log.Printf("[SPLIT] %d: \"%s\"", i+1, sentence)
		}
	}

	return finalSentences
}

// Enhance sentence for TTS
func enhanceSentenceForTTS(sentence string) string {
	enhanced := regexp.MustCompile(`[\r\n\t]+`).ReplaceAllString(sentence, " ")
	enhanced = regexp.MustCompile(`\s+`).ReplaceAllString(enhanced, " ")
	enhanced = strings.TrimSpace(enhanced)

	if enhanced == "" {
		return enhanced
	}

	// Check if sentence has ending punctuation
	hasEndingPunctuation := regexp.MustCompile(`[.!?…]$`).MatchString(enhanced)

	if !hasEndingPunctuation {
		// Add appropriate ending
		if strings.HasPrefix(enhanced, "¿") || regexp.MustCompile(`(?i)\b(qué|quién|cuándo|dónde|cómo|por qué|cuál)\b`).MatchString(enhanced) {
			enhanced += "?"
		} else if strings.HasPrefix(enhanced, "¡") || regexp.MustCompile(`(?i)\b(wow|increíble|excelente|fantástico)\b`).MatchString(enhanced) {
			enhanced += "!"
		} else {
			enhanced += "."
		}
	}

	// Add opening punctuation if missing
	if strings.HasSuffix(enhanced, "?") && !strings.Contains(enhanced, "¿") && !regexp.MustCompile(`(?i)\b(yes|no|si|sí)\b`).MatchString(enhanced) {
		enhanced = "¿" + enhanced
	}
	if strings.HasSuffix(enhanced, "!") && !strings.Contains(enhanced, "¡") && regexp.MustCompile(`(?i)\b(wow|increíble|excelente|fantástico|bravo|genial)\b`).MatchString(enhanced) {
		enhanced = "¡" + enhanced
	}

	// Remove duplicate punctuation
	enhanced = regexp.MustCompile(`¿¿+`).ReplaceAllString(enhanced, "¿")
	enhanced = regexp.MustCompile(`¡¡+`).ReplaceAllString(enhanced, "¡")
	enhanced = regexp.MustCompile(`\?\?+`).ReplaceAllString(enhanced, "?")
	enhanced = regexp.MustCompile(`!!+`).ReplaceAllString(enhanced, "!")

	return enhanced
}

// Split long sentences
func splitLongSentence(sentence string) []string {
	chunks := []string{}
	naturalBreaksPattern := regexp.MustCompile(`(?i)([,:;]\s+(?:pero|sin embargo|además|por tanto|por lo tanto|no obstante|mientras|cuando|donde|como|que|si|aunque|porque|ya que|dado que|puesto que))`)
	
	parts := naturalBreaksPattern.Split(sentence, -1)
	currentChunk := ""

	for _, part := range parts {
		if len(currentChunk) > 0 && len(currentChunk+part) > 200 {
			if strings.TrimSpace(currentChunk) != "" {
				chunks = append(chunks, enhanceSentenceForTTS(strings.TrimSpace(currentChunk)))
			}
			currentChunk = part
		} else {
			currentChunk += part
		}
	}

	if strings.TrimSpace(currentChunk) != "" {
		chunks = append(chunks, enhanceSentenceForTTS(strings.TrimSpace(currentChunk)))
	}

	if len(chunks) == 0 {
		return []string{sentence}
	}

	return chunks
}

// Merge short fragments
func mergeShortFragments(sentences []string) []string {
	merged := []string{}

	for i := 0; i < len(sentences); i++ {
		sentence := sentences[i]
		wordCount := len(regexp.MustCompile(`\b\w+\b`).FindAllString(sentence, -1))

		if wordCount < 4 && len(sentence) < 30 {
			if len(merged) > 0 {
				// Merge with previous
				merged[len(merged)-1] += " " + sentence
			} else if i+1 < len(sentences) {
				// Merge with next
				merged = append(merged, sentence+" "+sentences[i+1])
				i++ // Skip next
			} else {
				merged = append(merged, sentence)
			}
		} else {
			merged = append(merged, sentence)
		}
	}

	return merged
}

// Filter text segment with comprehensive processing
func filterTextSegment(textSegment string, modelReplacements [][]string) string {
	log.Printf("[FILTER] Processing segment: '%s'", truncateString(textSegment, 100))

	// Remove code blocks
	text := filterCodeBlocks(textSegment)
	log.Printf("[FILTER] After code block removal: '%s'", truncateString(text, 100))

	// Process line breaks
	text = processLineBreaks(text)
	log.Printf("[FILTER] After line break processing: '%s'", truncateString(text, 100))

	// Apply replacements
	if len(modelReplacements) > 0 {
		log.Printf("[FILTER] Using %d model-specific replacements from .onnx.json", len(modelReplacements))
		text = applyReplacements(text, modelReplacements)
	} else {
		log.Println("[FILTER] No model replacements found in .onnx.json - no replacements applied")
	}

	// Final cleanup
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	log.Printf("[FILTER] Final processed text: '%s'", truncateString(text, 100))
	return text
}

// Helper function to truncate strings for logging
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// Count words in text
func countWords(text string) int {
	return len(regexp.MustCompile(`\b\w+\b`).FindAllString(text, -1))
}

// Check if character is uppercase
func isUpperCase(r rune) bool {
	return unicode.IsUpper(r)
}
