import 'package:flutter/material.dart';
import 'package:speedplus/auth/shared/login_link.dart';
import 'package:speedplus/auth/widgets/registration_textfield.dart';
import 'package:speedplus/core/util/colors.dart';
import 'package:speedplus/core/util/sizes.dart';

class Login extends StatefulWidget {
  const Login({super.key});

  @override
  State<Login> createState() => _LoginState();
}

class _LoginState extends State<Login> {
  bool _isLoading = false;
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        automaticallyImplyLeading: true,
        title: const Text(
          "Create an account",
          style: TextStyle(fontWeight: FontWeight.bold, color: darkGreen),
        ),
        centerTitle: false,
      ),
      // bottomSheet: Column(
      //   children: [
      //     Expanded(
      //       child: ElevatedButton(
      //           onPressed: () {}, child: Text('Sign-Up')),
      //     ),
      //         LoginLink(),
      //   ],
      // ),
      body:   Stack(
        children: [
          SingleChildScrollView(
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: medium),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    "Complete the sign up process to get started",
                    style: Theme.of(context)
                        .textTheme
                        .bodyMedium
                        ?.copyWith(color: darkGreen),
                  ),
                  const SizedBox(height: small),

                  // Full Name
                  const RegistrationTextField(
                    label: "Full Name",
                    placeholder: "Enter full name here",
                  ),
                  const SizedBox(height: small),
                  const RegistrationTextField(
                    label: "Phone Number",
                    placeholder: "070764480655",
                  ),
                  const SizedBox(height: small),
                  const RegistrationTextField(
                    label: "Email Address",
                    placeholder: "nwaezeken@gmail.com",
                    isRequired: true,
                  ),
                  const SizedBox(height: small),
                  const RegistrationTextField(
                    label: "Password",
                    placeholder: "....................",
                    isRequired: true,
                  ),
                  const SizedBox(height: small),
                  const RegistrationTextField(
                    label: "Confirm Password",
                    placeholder: "....................",
                    isRequired: true,
                  ),
                  const SizedBox(height: small),
                  const RegistrationTextField(
                    label: "Referal/Promo Code (Optional)",
                    placeholder: "Obj2000",
                    isRequired: false,
                  ),
                  const SizedBox(height: small),

                  // login link
                  // const LoginLink(),
                ],
              ),
            ),
          ),

          // Loading overlay
          if (_isLoading)
            Container(
              color: Colors.black54, // Semi-transparent background
              width: double.infinity, // Fill the width
              height: double.infinity, // Fill the height
              child: const Center(
                child: Column(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    CircularProgressIndicator(),
                    Text('Verifying Your Details'),
                  ],
                ), // Loading indicator
              ),
            ),
        ],
      ),
    );
  }
}
