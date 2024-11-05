import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:speedplus/core/util/colors.dart';

class TermsAndPrivacy extends StatelessWidget {
  const TermsAndPrivacy({
    super.key,
  });

  @override
  Widget build(BuildContext context) {
    return RichText(
      text: TextSpan(
        children: [
          TextSpan(
            text: "By signing up with ",
            style: Theme.of(context)
                .textTheme
                .bodyMedium
                ?.copyWith(color: darkGrey),
          ),
          TextSpan(
            text: "Jetour, ",
            style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                color: primary, fontWeight: FontWeight.bold),
            recognizer: TapGestureRecognizer()..onTap = () {},
          ),
          TextSpan(
            text: "you agree to our ",
            style: Theme.of(context)
                .textTheme
                .bodyMedium
                ?.copyWith(color: darkGrey),
          ),
          TextSpan(
            text: "Terms of Service ",
            style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                color: primary, fontWeight: FontWeight.bold),
            recognizer: TapGestureRecognizer()..onTap = () {},
          ),
          TextSpan(
            text: "and ",
            style: Theme.of(context)
                .textTheme
                .bodyMedium
                ?.copyWith(color: darkGrey),
          ),
          TextSpan(
            text: "Privacy Policy.",
            style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                color: primary, fontWeight: FontWeight.bold),
            recognizer: TapGestureRecognizer()..onTap = () {},
          ),
        ],
      ),
    );
  }
}
