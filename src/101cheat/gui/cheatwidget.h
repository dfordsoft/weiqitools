#ifndef CheatWidget_H
#define CheatWidget_H

#include <QWidget>
#include <QMouseEvent>
#include <QPoint>
#include <QPixmap>
#include <QTimer>
#include <QSystemTrayIcon>

class CheatWidget : public QWidget
{
    Q_OBJECT

public:
    explicit CheatWidget(QWidget *parent = 0);
    ~CheatWidget();
protected:
    void mouseMoveEvent(QMouseEvent *event);
    void mousePressEvent(QMouseEvent *event);
    void mouseReleaseEvent(QMouseEvent *event);
    void paintEvent(QPaintEvent *event);    
    void moveEvent(QMoveEvent *event);
signals:

private slots:
    void trayIconActivated(QSystemTrayIcon::ActivationReason reason);
    void loadSkin();
    void showInFront();
    void quit();
    void clipboardChanged();
private:
    const int widgetMinWidth_ = 450;
    QPoint mouseMovePos_;
    QPixmap backgroundImage_;
    QPixmap leftPartBackgroundImage_;
    QPixmap midPartBackgroundImage_;
    QPixmap rightPartBackgroundImage_;
    QTimer* loadingAnimationTimer_;
    QSystemTrayIcon* trayIcon_;
    bool applySkin(const QString& skin);
    bool loadSkinConfiguration(const QString& configurationPath, QString& bgImagePath, QString& inputStyle, int& cutTop, int& cutBottom);
    bool loadSkinPackage(const QString& skinPath, QString& configurationPath);
};

#endif // CheatWidget_H
