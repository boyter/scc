module example::counter {
    use std::signer;

    /// A simple counter resource.
    struct Counter has key {
        value: u64,
    }

    /* outer
       /* nested */ block */
    public entry fun increment(account: &signer) acquires Counter {
        let addr = signer::address_of(account);
        let counter = borrow_global_mut<Counter>(addr);
        if (counter.value == 0) {
            counter.value = 1;
        } else {
            counter.value = counter.value + 1;
        };
    }

    public fun bounded_sum(n: u64): u64 {
        let i = 0;
        let total = 0;
        while (i < n) {
            if (i % 2 == 0 && i != 0) {
                total = total + i;
            };
            i = i + 1;
        };
        total
    }
}
