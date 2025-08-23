import appRouter from '@ohos.router';
import { AppConstants } from '../common/constants/AppConstants';
import { InfoListComponent } from '../view/InfoListComponent';

/**
 * Main Screen component.
 */
@Entry
@Component
struct MainScreen {
  private pageTitleRes: Resource = $r('app.string.main_title'); // set the title

  aboutToAppear() {
    if (appRouter.getParams()) {
      this.pageTitleRes = appRouter.getParams()[AppConstants.KEY_MAIN_SCREEN_DATA] as Resource;
    }
  }

  build() {
    Column() {
      Image($r('app.media.logo'))
        .width('80')
        .height('80')
        .borderRadius('50')
        .margin({ top: 100 })
      Column() {
        Button('Sign in')
          .width(300)
          .height(42)
          .borderRadius(50)
          .fontColor('#0078D4')
          .backgroundColor('#F0F0F0')
        Button('Try it now!')
          .width(300)
          .height(42)
          .borderRadius(50)
          .fontColor('#000000')
          .backgroundColor('transparent')
          .border({
            width: 1,
            color: "#000000"
          })
          .margin({
            top: 20
          })
          .onClick(() => {
            appRouter.pushUrl({
              url: AppConstants.HOME_PAGE
            });
            console.log('Clicked!');
          })
      }
      .justifyContent(FlexAlign.Center)
      .width('100%')
      .position({
        x: 0,
        y: '70%'
      })
    }
    .width('100%')
    .height(AppConstants.MAIN_COLUMN_HEIGHT_PERCENT)
    .backgroundColor('#FFFFF0')
  }
}
