import 'package:flutter/material.dart';
import 'package:smooth_page_indicator/smooth_page_indicator.dart';
import 'package:speedplus/core/util/colors.dart';
import 'package:speedplus/unboarding/unboardingItems.dart';

class Unboardingview extends StatefulWidget {
  const Unboardingview({super.key});

  @override
  State<Unboardingview> createState() => _UnboardingviewState();
}

class _UnboardingviewState extends State<Unboardingview> {
  final controller = Unboardingitems();
  final pageController = PageController();

  bool isLastPage = false;
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Column(
        children: [
          Expanded(
            child: PageView.builder(
                itemCount: controller.items.length,
                controller: pageController,
                itemBuilder: (context, index) {
                  return Column(
                    mainAxisAlignment: MainAxisAlignment.center,
                    children: [
                      Image.asset(controller.items[index].image),
                      const SizedBox(
                        height: 15,
                      ),
                      Text(
                        controller.items[index].title,
                        style: TextStyle(
                            fontSize: 30,
                            fontWeight: FontWeight.bold,
                            color: darkGreen),
                        textAlign: TextAlign.center,
                      ),
                      const SizedBox(
                        height: 15,
                      ),
                      Text(
                        style: TextStyle(fontSize: 20),
                        controller.items[index].description,
                        textAlign: TextAlign.center,
                      ),
                    ],
                  );
                }),
          ),
          Column(
            children: [
              SmoothPageIndicator(
                controller: pageController,
                count: controller.items.length,
                effect: const WormEffect(activeDotColor: darkGreen),
              ),
              Padding(
                padding:
                    const EdgeInsets.symmetric(horizontal: 15.0, vertical: 8),
                child: isLastPage? getStarted(context): Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    TextButton(
                        onPressed: () => pageController
                            .jumpToPage(controller.items.length - 1),
                        child: Text("Skip")),
                    TextButton(
                        style: TextButton.styleFrom(
                            backgroundColor: darkGreen, // foreground
                            ),
                        onPressed: () => pageController
                            .jumpToPage(controller.items.length + 1),
                        child: Text("Next", style: TextStyle(
                          color: yellow
                        ),)),
                  ],
                ),
              )
            ],
          )
        ],
      ),
    );
  }
}

Widget getStarted( context) {
  return Container(
      decoration: BoxDecoration(color: darkGreen),
      width: MediaQuery.of(context).size.width * .9,
      height: 55,
      child: TextButton(
          onPressed: () {
            
          },
          child: Text(
            "Get Started",
            style: TextStyle(color: yellow),
          )));
}
