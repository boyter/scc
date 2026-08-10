/* AL is the language of Microsoft Dynamics 365 Business Central.
   This block comment spans three lines and should be counted
   as three comment lines. */

namespace MyCompany.Sales;

using Microsoft.Foundation;

table 50100 "My Table"
{
    // a standalone line comment
    fields
    {
        field(1; Name; Text[50])
        {
            Caption = 'Name';
        }
    }

    procedure Compute(Value: Decimal): Decimal
    var
        Total: Decimal;
        Note: Text;
    begin
        Total := Value * 2;  // double it
        IF Total > 100 THEN  // legacy upper case AL
            Total := 100;
        Note := 'It''s at https://example.com';
        Note := 'not /* a comment */ either';
        Note := '// not a comment either';
        while Total > 200 do
            Total -= 1;
        exit(Total);
    end;
}
