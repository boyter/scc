(************************************************************************)
(*         *   The Coq Proof Assistant / The Coq Development Team       *)
(*  v      *   INRIA, CNRS and contributors - Copyright 1999-2018       *)
(* <O___,, *       (see CREDITS file for the list of authors)           *)
(*   \VV/  **************************************************************)
(*    //   *    This file is distributed under the terms of the         *)
(*         *     GNU Lesser General Public License Version 2.1          *)
(*         *     (see LICENSE file for the text of the license)         *)
(************************************************************************)

Require Export QArith.
Require Export Qreduction.

Hint Resolve Qlt_le_weak : qarith.

Definition Qabs (x:Q) := let (n,d):=x in (Z.abs n#d).

Lemma Qabs_case : forall (x:Q) (P : Q -> Type), (0 <= x -> P x) -> (x <= 0 -> P (- x)) -> P (Qabs x).
Proof.
intros x P H1 H2.
destruct x as [[|xn|xn] xd];
[apply H1|apply H1|apply H2];
abstract (compute; discriminate).
Defined.

Add Morphism Qabs with signature Qeq ==> Qeq as Qabs_wd.
intros [xn xd] [yn yd] H.
simpl.
unfold Qeq in *.
simpl in *.
change (Zpos yd)%Z with (Z.abs (Zpos yd)).
change (Zpos xd)%Z with (Z.abs (Zpos xd)).
repeat rewrite <- Z.abs_mul.
congruence.
Qed.

Lemma Qabs_pos : forall x, 0 <= x -> Qabs x == x.
Proof.
intros x H.
apply Qabs_case.
reflexivity.
intros H0.
setoid_replace x with 0.
reflexivity.
apply Qle_antisym; assumption.
Qed.

Lemma Qabs_neg : forall x, x <= 0 -> Qabs x == - x.
Proof.
intros x H.
apply Qabs_case.
intros H0.
setoid_replace x with 0.
reflexivity.
apply Qle_antisym; assumption.
reflexivity.
Qed.

Lemma Qabs_nonneg : forall x, 0 <= (Qabs x).
intros x.
apply Qabs_case.
auto.
apply (Qopp_le_compat x 0).
Qed.

Lemma Zabs_Qabs : forall n d, (Z.abs n#d)==Qabs (n#d).
Proof.
intros [|n|n]; reflexivity.
Qed.

Lemma Qabs_opp : forall x, Qabs (-x) == Qabs x.
Proof.
intros x.
do 2 apply Qabs_case; try (intros; ring);
(intros H0 H1;
setoid_replace x with 0;[reflexivity|];
apply Qle_antisym);try assumption;
rewrite Qle_minus_iff in *;
ring_simplify;
ring_simplify in H1;
assumption.
Qed.

Lemma Qabs_triangle : forall x y, Qabs (x+y) <= Qabs x + Qabs y.
Proof.
intros [xn xd] [yn yd].
unfold Qplus.
unfold Qle.
simpl.
apply Z.mul_le_mono_nonneg_r;auto with *.
change (Zpos yd)%Z with (Z.abs (Zpos yd)).
change (Zpos xd)%Z with (Z.abs (Zpos xd)).
repeat rewrite <- Z.abs_mul.
apply Z.abs_triangle.
Qed.

Lemma Qabs_Qmult : forall a b, Qabs (a*b) == (Qabs a)*(Qabs b).
Proof.
intros [an ad] [bn bd].
simpl.
rewrite Z.abs_mul.
reflexivity.
Qed.

Lemma Qabs_Qinv : forall q, Qabs (/ q) == / (Qabs q).
Proof.
  intros [n d]; simpl.
  unfold Qinv.
  case_eq n; intros; simpl in *; apply Qeq_refl.
Qed.

Lemma Qabs_Qminus x y: Qabs (x - y) = Qabs (y - x).
Proof.
 unfold Qminus, Qopp. simpl.
 rewrite Pos.mul_comm, <- Z.abs_opp.
 do 2 f_equal. ring.
Qed.

Lemma Qle_Qabs : forall a, a <= Qabs a.
Proof.
intros a.
apply Qabs_case; auto with *.
intros H.
apply Qle_trans with 0; try assumption.
change 0 with (-0).
apply Qopp_le_compat.
assumption.
Qed.

Lemma Qabs_triangle_reverse : forall x y, Qabs x - Qabs y <= Qabs (x - y).
Proof.
intros x y.
rewrite Qle_minus_iff.
setoid_replace (Qabs (x - y) + - (Qabs x - Qabs y)) with ((Qabs (x - y) + Qabs y) + - Qabs x) by ring.
rewrite <- Qle_minus_iff.
setoid_replace (Qabs x) with (Qabs (x-y+y)).
apply Qabs_triangle.
apply Qabs_wd.
ring.
Qed.

Lemma Qabs_Qle_condition x y: Qabs x <= y <-> -y <= x <= y.
Proof.
 split.
  split.
   rewrite <- (Qopp_opp x).
   apply Qopp_le_compat.
   apply Qle_trans with (Qabs (-x)).
   apply Qle_Qabs.
   now rewrite Qabs_opp.
  apply Qle_trans with (Qabs x); auto using Qle_Qabs.
 intros (H,H').
 apply Qabs_case; trivial.
 intros. rewrite <- (Qopp_opp y). now apply Qopp_le_compat.
Qed.

Lemma Qabs_diff_Qle_condition x y r: Qabs (x - y) <= r <-> x - r <= y <= x + r.
Proof.
 intros. unfold Qminus.
 rewrite Qabs_Qle_condition.
 rewrite <- (Qplus_le_l (-r) (x+-y) (y+r)).
 rewrite <- (Qplus_le_l (x+-y) r (y-r)).
 setoid_replace (-r + (y + r)) with y by ring.
 setoid_replace (r + (y - r)) with y by ring.
 setoid_replace (x + - y + (y + r)) with (x + r) by ring.
 setoid_replace (x + - y + (y - r)) with (x - r) by ring.
 intuition.
Qed.
