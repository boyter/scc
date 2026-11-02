// SPDX-License-Identifier: MIT

package processor

import (
	"testing"
)

// Perl reads any POD command paragraph at the start of a statement as the start
// of a POD block, which runs to =cut. Most POD opens with a heading rather than
// =pod, and only =pod was modelled, so documentation under =head1 and friends
// counted as code.
func TestCountStatsPerlPodHeadings(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Perl",
	}

	fileJob.SetContent(`package Foo;
use strict;

=head1 NAME

Foo - does things

=cut

sub new { return bless {}, shift }

=head2 name

Returns the name.

=cut

sub name { $_[0]{name} }
1;`)

	CountStats(&fileJob)

	if fileJob.Lines != 19 {
		t.Errorf("Expected 19 lines got %d", fileJob.Lines)
	}
	if fileJob.Code != 5 {
		t.Errorf("Expected 5 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 10 {
		t.Errorf("Expected 10 comments got %d", fileJob.Comment)
	}
	if fileJob.Blank != 4 {
		t.Errorf("Expected 4 blanks got %d", fileJob.Blank)
	}
}

func TestCountStatsPerlPod(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Perl",
	}

	fileJob.SetContent(`my $x = 1;
=pod

Notes.

=cut
print $x;`)

	CountStats(&fileJob)

	if fileJob.Code != 2 {
		t.Errorf("Expected 2 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 5 {
		t.Errorf("Expected 5 comments got %d", fileJob.Comment)
	}
	if fileJob.Blank != 0 {
		t.Errorf("Expected 0 blanks got %d", fileJob.Blank)
	}
}
