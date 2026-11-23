package llm

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/luongtran/luna/backend/internal/ai"
)

// ToeicGroups sends one request for new TOEIC question groups of one part (F27) and returns the JSON
// text; the service checks every group like an import before keeping it.
func (c *Client) ToeicGroups(ctx context.Context, req ai.ToeicGroupsRequest) (string, error) {
	return c.generate(ctx, "toeic_groups", toeicPrompt(req), toeicSchema, 0.6,
		slog.Int("part", req.Part), slog.Int("count", req.Count), slog.Int("questions", req.Questions))
}

// toeicLevels describes what a group at each target score is like, so the AI calibrates the
// vocabulary, the sentences and how far the right option paraphrases the material.
var toeicLevels = map[int]string{
	450: "TOEIC 450 (elementary): high-frequency everyday and workplace words, short sentences, mostly simple tenses. " +
		"The answer is stated almost word for word; distractors are clearly wrong once the material is understood.",
	600: "TOEIC 600 (intermediate): common business vocabulary, compound sentences, some phrasal verbs. " +
		"The right option lightly paraphrases the material; one distractor repeats a word from the material as a trap.",
	750: "TOEIC 750 (upper-intermediate): wider business vocabulary and collocations, complex sentences, relative and " +
		"conditional clauses. The right option paraphrases (synonyms, a changed structure); some questions need a simple " +
		"inference; distractors are plausible and mention things that really appear in the material.",
	900: "TOEIC 900 (advanced): low-frequency but realistic business vocabulary, idiomatic and indirect language, dense " +
		"sentences. Answers need inference, implied meaning or combining two pieces of information; distractors are " +
		"close in meaning and reuse the material's words; only careful reading or listening separates them.",
}

// toeicPartSpecs is the specification of one group of each part (index = part). Part 7's has a %d
// for the number of questions.
var toeicPartSpecs = [8]string{
	"",
	"PART 1 - PHOTOGRAPHS. Each group is ONE question: \"scene\" + 4 spoken statements as \"options\"; no \"prompt\", no \"script\".\n" +
		"- \"scene\": a precise English description of ONE realistic photograph (who is there, what each person is doing, the objects, " +
		"where they are, their positions and states), 25-60 words. It is used to generate the photo, so it must be fully visual: " +
		"no readable text, signs, logos or brand names, nothing that cannot be seen.\n" +
		"- Mix photos of people (one person or several) and photos of objects or places with no people.\n" +
		"- The key describes the photo exactly: an action in progress (\"is ...ing\"), a state (\"has been ...\", \"is being ...\", " +
		"\"are stacked\"), a position (\"next to\", \"on top of\").\n" +
		"- Distractors: a wrong action by the right person, an object that is not in the photo, a word that sounds like one in " +
		"the photo (copy / coffee), the right object in the wrong state or place, a statement about people when there are none.\n" +
		"- The four statements have the same kind of subject and similar length (5-10 words). \"translateVi\" gives the four " +
		"statements in Vietnamese as \"(A) ... (B) ... (C) ... (D) ...\".",
	"PART 2 - QUESTION-RESPONSE. Each group is ONE question: a short spoken question or statement + 3 spoken responses.\n" +
		"- \"script\": one turn with the spoken question or statement (speaker \"m\" or \"w\", any accent); \"prompt\" repeats " +
		"that exact text.\n" +
		"- Spread the item types: WH questions (who, what, when, where, why, how), yes/no, choice (A or B?), requests and " +
		"offers, statements, and indirect answers.\n" +
		"- The key is a natural reply and often answers indirectly (\"I haven't heard yet.\", \"Let me check the schedule.\", " +
		"\"Mr. Kim is handling that.\").\n" +
		"- Distractors: repeat a word from the question or a similar-sounding word, answer a different WH word (a place for " +
		"\"when\"), give Yes/No to a WH question, or answer a related but different question.\n" +
		"- Responses have 3-10 words each and similar length. \"translateVi\" gives the question and the three responses in Vietnamese.",
	"PART 3 - CONVERSATIONS. Each group is one conversation + exactly 3 questions of 4 options.\n" +
		"- \"script\": 5-8 turns, 90-130 words, between 2 speakers (\"m\", \"w\") or, in about one group in four, 3 speakers " +
		"(\"m\", \"w\", \"m2\" or \"w2\"). Natural spoken English with contractions. Each speaker keeps one accent; the speakers " +
		"of a group have different accents (us, gb, au, ca). Every turn has a Vietnamese \"translateVi\".\n" +
		"- Questions follow the order in which the information is heard: 1) gist or who/where/why (\"Where most likely are the " +
		"speakers?\", \"What is the conversation mainly about?\"), 2) a detail, 3) a next action, request or suggestion (\"What " +
		"will the woman most likely do next?\").\n" +
		"- About one group in three has an implied-meaning question: 'Why does the man say, \"...\"?' or 'What does the woman " +
		"imply when she says, \"...\"?', quoting words copied exactly from a turn.\n" +
		"- About one group in four has a \"graphic\" (kind table, schedule, price-list, floor-plan or bar-chart, with a title) and " +
		"one question beginning \"Look at the graphic.\" whose options are values from the graphic and whose answer needs both " +
		"the conversation and the graphic. A group without such a question has no \"graphic\".\n" +
		"- Options are short (2-8 words), in parallel form, as printed in the test book.",
	"PART 4 - TALKS. Each group is one talk by ONE speaker + exactly 3 questions of 4 options.\n" +
		"- Talk types: announcements, telephone or voicemail messages, advertisements, tours, meeting excerpts, news or traffic " +
		"reports, speeches and introductions, workshops.\n" +
		"- \"script\": 100-150 words split into 4-7 turns of the SAME speaker and accent (1-3 sentences each), natural spoken " +
		"English with contractions. \"speaker\" is the code \"m\" or \"w\", never a name or a job title. Every turn has a " +
		"Vietnamese \"translateVi\".\n" +
		"- Questions: 1) purpose, topic, speaker or listeners, place, 2) a detail, 3) what the listeners should do or what will " +
		"happen next. Implied-meaning and graphic questions as in Part 3, in about one group in three and one group in four.",
	"PART 5 - INCOMPLETE SENTENCES. Each group is ONE question: one business sentence (12-25 words) in \"prompt\" with exactly " +
		"one blank \"___\", and 4 options.\n" +
		"- Test grammar (word form, verb tense, voice or form, pronoun, preposition, conjunction, relative, comparison) or " +
		"vocabulary (meaning, collocation). Spread the types over the groups.\n" +
		"- Grammar items: the 4 options are forms of the same word or members of one closed class (efficient / efficiently / " +
		"efficiency / efficiencies). Vocabulary items: 4 words of the same part of speech and similar length; only one fits the " +
		"meaning and the collocation.\n" +
		"- The sentence gives enough context for exactly one option; check every other option in the blank.\n" +
		"- \"translateVi\" is the Vietnamese of the completed sentence.",
	"PART 6 - TEXT COMPLETION. Each group is ONE text + exactly 4 questions of 4 options.\n" +
		"- \"docs\": one text (email, letter, memo, notice, article, advertisement or web page) of 120-180 words: \"kind\", " +
		"\"title\", \"meta\" lines (From, To, Subject, Date as fitting) and paragraphs of kind \"p\". The body contains the blanks " +
		"[[1]] [[2]] [[3]] [[4]] in order, each exactly once; question n is blank [[n]] and has no \"prompt\".\n" +
		"- Mix: 1-2 grammar blanks (verb tense across the text, word form, pronoun reference), 1 vocabulary blank decided by the " +
		"context, 1 connector blank (however, therefore, in addition...) and exactly 1 sentence-insertion blank whose 4 options " +
		"are complete sentences, only one of which fits the logic of the paragraph.\n" +
		"- At least one blank needs information from another sentence of the text, not only its own sentence.\n" +
		"- The text has a Vietnamese \"translateVi\" (of the completed text).",
	"PART 7 - READING COMPREHENSION. Write groups whose questions add up to EXACTLY %d, with 1 to 3 texts and 2 to 5 " +
		"questions per group.\n" +
		"- Text kinds: email, letter, memo, notice, ad, article, web, invoice, form, schedule, chat, review. Blocks of kind " +
		"\"p\", \"h\", \"list\" (with \"items\"), \"table\" (with \"table\": columns and rows) or \"msg\" (a chat line with \"who\" " +
		"and \"time\"). A single text has 80-200 words; each text of a multiple-text group 60-150 words.\n" +
		"- Include at least one group with two or three texts in which at least one question can only be answered by combining " +
		"two texts (tag cross-reference).\n" +
		"- Question types: main purpose, detail, NOT true (\"What is NOT mentioned...\"), inference (\"What is suggested...\"), " +
		"vocabulary in context ('In paragraph 1, line 2, the word \"...\" is closest in meaning to'), and implied meaning in a " +
		"chat ('At 10:15 A.M., what does Ms. Lee mean when she writes, \"...\"?').\n" +
		"- Each text has a Vietnamese \"translateVi\"; each question has a Vietnamese \"translateVi\" of its prompt.",
}

// toeicPromptRules are the parts of the prompt every request shares: the item-writing rules, the
// Vietnamese explanations, the evidence, the format and the final check.
const toeicPromptRules = `ITEM-WRITING RULES (every question)
1. Exactly one key. Read every distractor against the material: if a careful test taker could defend it, rewrite it.
2. Distractors are plausible: each one targets a specific misunderstanding (a word heard or read in the material used in the wrong way, a similar-sounding word, a detail about the wrong person, time or place, a wrong inference, a grammatically possible but wrong form).
3. Options are parallel: same grammatical form, similar length and level of detail; the key is not the longest or the most precise option. No "all/none of the above", no options that overlap or contradict only each other.
4. At 600 and above, the key paraphrases the material instead of repeating its words; word-for-word repetition belongs to the distractors.
5. "answer" is the index of the key (0 = A). Spread the keys evenly over the positions across the groups.
6. "no" counts from 1 inside each group.
7. Explanations, evidence and quotes refer only to words that are really in the material: never cite a phrase, a fact or a sentence that is not there. Write the material first, then the questions, then check each explanation against the material.
8. Part 6/7 texts: header lines (From, To, Subject, Date, URL) go in "meta", never in "body"; the body holds only the text itself. In Part 6 the words around each blank must make it answerable: keep the whole sentence (e.g. "In an effort to [[1]] the campaign, we ...").

EXPLANATIONS AND TRANSLATIONS (in Vietnamese)
- "explainVi": first say why the key is right, quoting or pointing to the words of the material; then one short reason for each wrong option, naming its trap (repeated word, similar sound, wrong person/time/place, inference too far, wrong word form...). 2-5 sentences, natural Vietnamese with full diacritics.
- "translateVi": faithful and natural. Translate numbers, dates, times, frequencies and quantities exactly (annual = "hằng năm", weekly = "hằng tuần", quarterly = "hằng quý"); never add or drop information.
- "vocab": 1-3 words or phrases worth learning from the group, each with an IPA transcription and a short Vietnamese meaning.

EVIDENCE (Part 3, 4, 6 and 7)
- "evidence" lists the words that give the answer away, copied EXACTLY (same spelling and punctuation) from the material: for a script turn, doc -1 and "turn" = the index of the turn from 0; for a text, "doc" = the index of the text from 0 and "block" = the index of the block in its body from 0. Use short spans of 3-15 words.

FORMAT
- Return JSON that matches the schema; leave out the fields the part does not use. Do not write "key" or "part": the server sets them.
- Accents are "us", "gb", "au" or "ca". Plain text only; inside Part 6/7 texts **bold** is allowed.

FINAL CHECK (do it silently before answering)
- Language: the material, questions and options are in English; only explainVi, translateVi and meaningVi are in Vietnamese.
- Count: the number of groups and questions is exactly what the task asks; each group has the number of questions and options its part requires.
- Each question has one and only one defensible answer, and the keys are spread over the positions.
- Every evidence span appears exactly in the material at the index given; every sentence quoted in a question appears exactly in the material.
- Level: vocabulary, sentence length and paraphrase match the target level.
- Vietnamese: explanations and translations are accurate, natural and fully accented.
`

func toeicPrompt(req ai.ToeicGroupsRequest) string {
	part := req.Part
	if part < 1 || part > 7 {
		part = 5
	}
	level, ok := toeicLevels[req.Level]
	if !ok {
		level = toeicLevels[600]
	}
	spec := toeicPartSpecs[part]
	task := fmt.Sprintf("Write exactly %d new groups of Part %d.", req.Count, part)
	if part == 7 {
		spec = fmt.Sprintf(spec, req.Questions)
		task = fmt.Sprintf("Write new Part 7 groups with exactly %d questions in total.", req.Questions)
	}

	var b strings.Builder
	b.WriteString("ROLE\nYou are a senior item writer and editor for the TOEIC Listening and Reading test, with long experience " +
		"writing official-style practice tests. You write for Vietnamese learners: the test material is in English; every " +
		"explanation and translation is in natural Vietnamese with full diacritics.\n\n")
	fmt.Fprintf(&b, "TASK\n%s The material must be entirely original: never copy or adapt ETS tests, published books or websites.\n\n", task)
	b.WriteString("LANGUAGE (strict)\n" +
		"- ENGLISH ONLY for everything a test taker reads or hears: \"scene\", every script \"text\", \"prompt\", \"options\", " +
		"the \"graphic\", and every part of a text in \"docs\" (\"title\", \"meta\" keys and values such as From / To / " +
		"Subject / Date, every block of \"body\"), and \"word\" in \"vocab\".\n" +
		"- VIETNAMESE ONLY for \"explainVi\", every \"translateVi\" and \"meaningVi\".\n" +
		"- A single Vietnamese word in an English field makes the group unusable.\n\n")
	fmt.Fprintf(&b, "TARGET LEVEL\n%s Set \"level\" to %d in every group.\n\n", level, req.Level)
	b.WriteString("CONTENT\n" +
		"- Real-world workplace and everyday business situations: offices, meetings, hiring and training, sales and marketing, " +
		"shopping and orders, deliveries and shipping, finance and banking, travel, hotels and restaurants, events and " +
		"conferences, maintenance and facilities, safety at work.\n" +
		"- Fictional people with internationally varied names; fictional companies, products, addresses, websites, phone numbers " +
		"and prices. No real brands or famous people.\n" +
		"- Avoid politics, religion, disasters, illness, violence and anything that needs specialist or cultural knowledge. " +
		"Every answer must come from the material alone.\n" +
		"- Vary the situations, names and question types across the groups; do not repeat a situation the part already has " +
		"(listed at the end).\n\n")
	b.WriteString(spec + "\n\n")
	fmt.Fprintf(&b, "QUESTION TYPES\n\"tags\": one or two codes from this list only: %s.\n\n", tagList(req.Tags))
	b.WriteString(toeicPromptRules)
	if t := strings.TrimSpace(req.Topic); t != "" {
		fmt.Fprintf(&b, "\nADMIN'S FOCUS (follow it as long as it fits the part and the rules): %s\n", t)
	}
	if len(req.Existing) > 0 {
		b.WriteString("\nSITUATIONS AND QUESTIONS THE PART ALREADY HAS (write different ones; JSON):\n")
		writeJSONLine(&b, req.Existing)
	}
	return b.String()
}

func tagList(tags []ai.ToeicTag) string {
	parts := make([]string, len(tags))
	for i, t := range tags {
		parts[i] = fmt.Sprintf("%s (%s)", t.Code, t.Label)
	}
	return strings.Join(parts, ", ")
}
