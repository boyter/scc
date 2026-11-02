// SPDX-License-Identifier: MIT

package processor

import (
	"testing"
)

// Twig writes a comment as {# ... #}, and its lexer drops everything from the
// {# to the first #} after it, newlines included. Comments do not nest. Twig
// Template declared no comment at all, so a comment line, and every line of
// the docblock that opens many Twig templates, counted as code, and a tag that
// had been commented out still counted towards complexity.

func TestCountStatsTwigCommentOnItsOwnLine(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Twig Template",
	}

	fileJob.SetContent(`{% extends "base.html.twig" %}

{# The title block is filled in by each page. #}
{% block title %}
  {#- whitespace control trims around the comment -#}
  {{ page.title }}
  {#~ and this one trims up to the line ~#}
{% endblock %}`)

	CountStats(&fileJob)

	if fileJob.Lines != 8 {
		t.Errorf("Expected 8 lines got %d", fileJob.Lines)
	}
	if fileJob.Code != 4 {
		t.Errorf("Expected 4 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 3 {
		t.Errorf("Expected 3 comments got %d", fileJob.Comment)
	}
	if fileJob.Blank != 1 {
		t.Errorf("Expected 1 blank got %d", fileJob.Blank)
	}
}

func TestCountStatsTwigMultiLineComment(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Twig Template",
	}

	fileJob.SetContent(`{#
/**
 * @file
 * Displays the title of a page.
 *
 * Available variables:
 * - title: The page title.
 */
#}
{% if title %}
  <h1>{{ title }}</h1>
{% endif %}`)

	CountStats(&fileJob)

	if fileJob.Lines != 12 {
		t.Errorf("Expected 12 lines got %d", fileJob.Lines)
	}
	if fileJob.Code != 3 {
		t.Errorf("Expected 3 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 9 {
		t.Errorf("Expected 9 comments got %d", fileJob.Comment)
	}
	if fileJob.Blank != 0 {
		t.Errorf("Expected 0 blanks got %d", fileJob.Blank)
	}
	if fileJob.Complexity != 1 {
		t.Errorf("Expected complexity 1 got %d", fileJob.Complexity)
	}
}

func TestCountStatsTwigCommentedOutTagIsNotComplexity(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Twig Template",
	}

	fileJob.SetContent(`{#
{% if user.isAdmin %}
  <a href="/admin">Admin</a>
{% endif %}
#}
<p>{{ greeting }}</p>`)

	CountStats(&fileJob)

	if fileJob.Code != 1 {
		t.Errorf("Expected 1 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 5 {
		t.Errorf("Expected 5 comments got %d", fileJob.Comment)
	}
	if fileJob.Complexity != 0 {
		t.Errorf("Expected complexity 0 got %d", fileJob.Complexity)
	}
}

func TestCountStatsTwigCommentNextToMarkupIsCode(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Twig Template",
	}

	fileJob.SetContent(`<a href="#top">{{ 'Back to top'|trans }}</a> {# shown on every page #}
{# label #}<h2>{{ label }}</h2>
<p style="color: #333">{{ count }}</p>`)

	CountStats(&fileJob)

	if fileJob.Lines != 3 {
		t.Errorf("Expected 3 lines got %d", fileJob.Lines)
	}
	if fileJob.Code != 3 {
		t.Errorf("Expected 3 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 0 {
		t.Errorf("Expected 0 comments got %d", fileJob.Comment)
	}
}
