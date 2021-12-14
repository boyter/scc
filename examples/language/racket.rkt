#lang racket/base

(require racket/private/norm-arity)

(provide normalize-arity normalized-arity? arity=? arity-includes?)

(define (normalized-arity? a)
  (or (null? a)
      (arity? a)
      (and (list? a)
           ((length a) . >= . 2)
           (andmap arity? a)
           (if (ormap arity-at-least? a)
               (non-empty-non-singleton-sorted-list-ending-with-arity? a)
               (non-singleton-non-empty-sorted-list? a)))))

(define (arity? a)
  (or (exact-nonnegative-integer? a)
      (and (arity-at-least? a)
           (exact-nonnegative-integer? (arity-at-least-value a)))))

;; non-empty-non-singleton-sorted-list-ending-with-arity? : xx -> boolean
;; know that 'a' is a list of at least 2 elements
(define (non-empty-non-singleton-sorted-list-ending-with-arity? a)
  (let loop ([bound (car a)]
             [lst (cdr a)])
    (cond
      [(null? (cdr lst))
       (and (arity-at-least? (car lst))
            (> (arity-at-least-value (car lst)) (+ 1 bound)))]
      [else
       (and (exact-nonnegative-integer? (car lst))
            ((car lst) . > . bound)
            (loop (car lst)
                  (cdr lst)))])))

(define (non-empty-sorted-list? a)
  (and (pair? a)
       (sorted-list? a)))

(define (non-singleton-non-empty-sorted-list? a)
  (and (pair? a)
       (pair? (cdr a))
       (sorted-list? a)))

(define (sorted-list? a)
  (or (null? a)
      (sorted/bounded-list? (cdr a) (car a))))

(define (sorted/bounded-list? a bound)
  (or (null? a)
      (and (number? (car a))
           (< bound (car a))
           (sorted/bounded-list? (cdr a) (car a)))))

(define (arity-supports-number? arity n)
  (cond
    [(exact-nonnegative-integer? arity) (= arity n)]
    [(arity-at-least? arity) (<= (arity-at-least-value arity) n)]
    [(list? arity)
     (for/or {[elem (in-list arity)]}
       (arity-supports-number? elem n))]))

(define (arity-supports-at-least? arity n)
  (cond
    [(exact-nonnegative-integer? arity) #f]
    [(arity-at-least? arity) (<= (arity-at-least-value arity) n)]
    [(list? arity)
     (define min-at-least
       (for/fold {[min-at-least #f]} {[elem (in-list arity)]}
         (cond
           [(exact-nonnegative-integer? elem) min-at-least]
           [(arity-at-least? elem)
            (cond
              [(not min-at-least) (arity-at-least-value elem)]
              [else (min min-at-least (arity-at-least-value elem))])])))
     (cond
       [(not min-at-least) #f]
       [else
        (for/and {[i (in-range n min-at-least)]}
          (arity-supports-number? arity i))])]))

(define (unchecked-arity-includes? one two)
  (cond
    [(exact-nonnegative-integer? two)
     (arity-supports-number? one two)]
    [(arity-at-least? two)
     (arity-supports-at-least? one (arity-at-least-value two))]
    [(list? two)
     (for/and {[elem (in-list two)]}
       (unchecked-arity-includes? one elem))]))

(define (arity-includes? one two)
  (unless (procedure-arity? one)
    (raise-argument-error 'arity-includes? "procedure-arity?" 0 one two))
  (unless (procedure-arity? two)
    (raise-argument-error 'arity-includes? "procedure-arity?" 1 one two))
  (unchecked-arity-includes? one two))

(define (arity=? one two)
  (unless (procedure-arity? one)
    (raise-argument-error 'arity=? "procedure-arity?" 0 one two))
  (unless (procedure-arity? two)
    (raise-argument-error 'arity=? "procedure-arity?" 1 one two))
  (and
    (unchecked-arity-includes? one two)
    (unchecked-arity-includes? two one)))