import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:speedplus/core/util/colors.dart';
import 'package:speedplus/core/util/sizes.dart';

class RegistrationTextField extends StatelessWidget {
  const RegistrationTextField({
    super.key,
    required this.label,
    required this.placeholder,
    this.isRequired = true,
  });

  final String label, placeholder;
  final bool? isRequired;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        RichText(
          text: TextSpan(
            children: [
              TextSpan(
                text: label,
                style: Theme.of(context)
                    .textTheme
                    .labelMedium
                    ?.copyWith(color: black, fontSize: smallMedium),
              ),
              if (isRequired == true)
                TextSpan(
                  text: " *",
                  style: Theme.of(context)
                      .textTheme
                      .labelLarge
                      ?.copyWith(color: red, fontSize: smallMedium),
                ),
            ],
          ),
        ),
        const SizedBox(height: extraSmall),
        TextField(
          decoration: InputDecoration(
            label: Text(
              placeholder,
              style: context.textTheme.labelLarge?.copyWith(color: lightGrey),
            ),
            enabledBorder: OutlineInputBorder(
              borderSide: const BorderSide(color: lightGrey),
              borderRadius: BorderRadius.circular(large),
            ),
            focusedBorder: OutlineInputBorder(
              borderSide: const BorderSide(color: darkGreen),
              borderRadius: BorderRadius.circular(large),
            ),
          ),
        ),
      ],
    );
  }
}

class _VerifyingState extends StatefulWidget {
  const _VerifyingState();

  @override
  State<_VerifyingState> createState() => __VerifyingStateState();
}

class __VerifyingStateState extends State<_VerifyingState> {
  @override
  Widget build(BuildContext context) {
    return const Center();
  }
}
