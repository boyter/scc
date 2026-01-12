/**
 * Example Apex class demonstrating language features
 * for scc code counting validation
 */
public class AccountHandler {
    // Single line comment
    private static final String DEFAULT_STATUS = 'Active';

    public static List<Account> getActiveAccounts() {
        List<Account> accounts = [SELECT Id, Name FROM Account WHERE Status__c = :DEFAULT_STATUS];

        if (accounts.isEmpty()) {
            return new List<Account>();
        }

        for (Account acc : accounts) {
            if (acc.Name != null && acc.Name.length() > 0) {
                System.debug('Processing: ' + acc.Name);
            }
        }

        return accounts;
    }

    public static void processAccount(Account acc) {
        try {
            update acc;
        } catch (DmlException e) {
            System.debug('Error: ' + e.getMessage());
        } finally {
            System.debug('Processing complete');
        }
    }

    /*
     * Multi-line comment block
     * demonstrating block comments
     */
    public static Integer calculateTotal(List<Integer> values) {
        Integer total = 0;
        Integer i = 0;
        while (i < values.size()) {
            total += values[i];
            i++;
        }
        return total;
    }
}
