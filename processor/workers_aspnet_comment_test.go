// SPDX-License-Identifier: MIT

package processor

import (
	"testing"
)

// ASP.NET declared its server side comment as <%-- ... --> so the opener never
// closed on the real --%> terminator and the scanner ran the comment to EOF,
// swallowing the rest of the markup as comment.

func TestCountStatsAspNetServerCommentCloses(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "ASP.NET",
	}

	fileJob.SetContent(`<%@ Page Language="C#" Inherits="Demo.Default" %>
<%-- server side comment, never sent to the browser --%>
<html>
<body>
    <form id="form1" runat="server">
        <asp:Label ID="Label1" runat="server" Text="Hello" />
    </form>
</body>
</html>`)

	CountStats(&fileJob)

	if fileJob.Lines != 9 {
		t.Errorf("Expected 9 lines got %d", fileJob.Lines)
	}
	if fileJob.Code != 8 {
		t.Errorf("Expected 8 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 1 {
		t.Errorf("Expected 1 comment got %d", fileJob.Comment)
	}
	if fileJob.Blank != 0 {
		t.Errorf("Expected 0 blanks got %d", fileJob.Blank)
	}
}

func TestCountStatsAspNetMultiLineServerComment(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "ASP.NET",
	}

	fileJob.SetContent(`<%@ Control Language="C#" %>
<%--
 disabled for now
 restore before release
--%>

<asp:Panel ID="Panel1" runat="server" />`)

	CountStats(&fileJob)

	if fileJob.Lines != 7 {
		t.Errorf("Expected 7 lines got %d", fileJob.Lines)
	}
	if fileJob.Code != 2 {
		t.Errorf("Expected 2 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 4 {
		t.Errorf("Expected 4 comments got %d", fileJob.Comment)
	}
	if fileJob.Blank != 1 {
		t.Errorf("Expected 1 blank got %d", fileJob.Blank)
	}
}

func TestCountStatsAspNetHtmlCommentStillCounts(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "ASP.NET",
	}

	fileJob.SetContent(`<html>
<!-- plain html comment
 over two lines -->
<body>
    <p>text</p>
</body>
</html>`)

	CountStats(&fileJob)

	if fileJob.Lines != 7 {
		t.Errorf("Expected 7 lines got %d", fileJob.Lines)
	}
	if fileJob.Code != 5 {
		t.Errorf("Expected 5 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 2 {
		t.Errorf("Expected 2 comments got %d", fileJob.Comment)
	}
	if fileJob.Blank != 0 {
		t.Errorf("Expected 0 blanks got %d", fileJob.Blank)
	}
}
