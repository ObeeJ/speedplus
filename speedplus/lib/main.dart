import 'package:flutter/material.dart';
import 'package:speedplus/auth/registrationSceen.dart';
import 'package:speedplus/core/util/colors.dart';
// import 'package:speedplus/unboarding/unboardingView.dart';

void main() {
  runApp(const MyApp());
}

class MyApp extends StatelessWidget {
  const MyApp({super.key});

  // This widget is the root of your application.
  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      debugShowCheckedModeBanner: false,
      title: 'SpeedPlus',
      theme: ThemeData(
      
        colorScheme: ColorScheme.fromSeed(seedColor: darkGreen),
        useMaterial3: true,
      ),
      home:const Login(),
    );
  }
}

