#include "stdafx.h"
#include <JlCompress.h>
#include <QRegularExpression>
#include "uglobalhotkeys.h"
#include "cheatwidget.h"

CheatWidget::CheatWidget(QWidget *parent) :
    QWidget(parent),
    mouseMovePos_(0, 0)
{
#if defined(Q_OS_WIN)
    setWindowFlags(Qt::FramelessWindowHint | Qt::Tool | Qt::WindowStaysOnTopHint);
#else
    setWindowFlags(Qt::FramelessWindowHint | Qt::WindowStaysOnTopHint);
#endif
    setAttribute(Qt::WA_InputMethodEnabled);
    setAttribute(Qt::WA_TranslucentBackground);

    setFocusPolicy(Qt::ClickFocus);
    
    if (!applySkin(":/skins/mercury_wide.cheatskin"))
    {
        qCritical() << "loading skin failed";
        return;
    }
#ifdef Q_WS_MAC
    QMacStyle::setFocusRectPolicy(input, QMacStyle::FocusDisabled);
#endif

    QAction *quitAction = new QAction(tr("E&xit"), this);
    quitAction->setShortcut(tr("Ctrl+Q"));
    connect(quitAction, &QAction::triggered, this, &CheatWidget::quit);
    addAction(quitAction);

    QAction *loadSkinAction = new QAction(tr("Load &Skin"), this);
    loadSkinAction->setShortcut(tr("Ctrl+O"));
    connect(loadSkinAction, &QAction::triggered, this, &CheatWidget::loadSkin);
    addAction(loadSkinAction);

    setContextMenuPolicy(Qt::ActionsContextMenu);

    QAction *showAction = new QAction(tr("Show"), this);
    showAction->setShortcut(tr("Ctrl+Alt+Space"));
    connect(showAction, &QAction::triggered, this, &CheatWidget::showInFront);

    QMenu* trayiconMenu = new QMenu(this);
    trayiconMenu->addAction(showAction);
    trayiconMenu->addAction(loadSkinAction);
    trayiconMenu->addAction(quitAction);
    trayIcon_ = new QSystemTrayIcon(this);
    connect(trayIcon_, &QSystemTrayIcon::activated, this, &CheatWidget::trayIconActivated);
    trayIcon_->setContextMenu(trayiconMenu);
    trayIcon_->setIcon(QIcon(":/cheat.ico"));
    trayIcon_->setToolTip(tr("101Cheat - Show answer of 101weiqi.com's puzzles!"));
    trayIcon_->show();

    connect(QApplication::clipboard(), &QClipboard::dataChanged, this, &CheatWidget::clipboardChanged);
}

CheatWidget::~CheatWidget()
{
}

void CheatWidget::mouseMoveEvent(QMouseEvent *event)
{
    if ((event->buttons() & Qt::LeftButton))
    {
        move(mapToParent(event->pos() - mouseMovePos_));
    }
}

void CheatWidget::mousePressEvent(QMouseEvent *event)
{
    if (event->button() == Qt::LeftButton)
    {
        mouseMovePos_ = event->pos();
    }
}

void CheatWidget::mouseReleaseEvent(QMouseEvent* event)
{
    if (event->button() == Qt::LeftButton)
    {
        mouseMovePos_ = QPoint(0, 0);
    }
}

void CheatWidget::paintEvent(QPaintEvent* event)
{
    QStyleOption styleOption;
    styleOption.init(this);
    QPainter painter(this);
    painter.setRenderHint(QPainter::Antialiasing);
    style()->drawPrimitive(QStyle::PE_Widget, &styleOption, &painter, this);
    QSize size(backgroundImage_.size());

    if (size.width() > widgetMinWidth_)
        painter.drawPixmap(0, 0, backgroundImage_);
    else
    {
        painter.drawPixmap(0, 0, leftPartBackgroundImage_);
        painter.drawPixmap(size.width() / 2 - 1, 0, midPartBackgroundImage_);
        painter.drawPixmap(widgetMinWidth_ - (size.width() / 2 - 1), 0, rightPartBackgroundImage_);
    }

    QPen pen(painter.pen());
    pen.setColor(0xFFFFFF);
    painter.setPen(pen);

    int i = 0;
    for( const QString answer : answer_)
        painter.drawText(40, 40 + 15 * i++, answer);
    QWidget::paintEvent(event);
}

void CheatWidget::moveEvent(QMoveEvent* /*event*/)
{
}

void CheatWidget::trayIconActivated(QSystemTrayIcon::ActivationReason reason)
{
    switch(reason)
    {
    case QSystemTrayIcon::DoubleClick:
        if (isHidden())
            show();
        activateWindow();
        raise();
        break;
    default:
        break;
    }
}

void CheatWidget::loadSkin()
{
    QString fileName = QFileDialog::getOpenFileName(this,
        tr("Load 101Cheat Skin"),
        "",
        tr("101Cheat Skin File (*.cheatskin);;101Cheat Skin Configuration (*.xml);;All files (*.*)"));
    if (fileName.isEmpty())
        return;
    applySkin(fileName);
}

void CheatWidget::showInFront()
{
    if (isHidden())
        show();
    activateWindow();
    raise();
}

void CheatWidget::quit()
{
    qApp->quit();
}

void CheatWidget::clipboardChanged()
{
    QClipboard *clipboard = QApplication::clipboard();
    QString originalText = clipboard->text();
    // get answers
    QString pattern("var pos_[0-9]+\\s?=\\s?\\[([^\\]]+)\\];ans.push\\(pos_[0-9]+\\);an_isoks\\.push\\(([0-9])\\)");

    QRegularExpression re(pattern,
                          QRegularExpression::MultilineOption | QRegularExpression::DotMatchesEverythingOption);
    QRegularExpressionMatchIterator it = re.globalMatch(originalText);

    QStringList answer;
    while (it.hasNext())
    {
        QRegularExpressionMatch match = it.next();
        QString steps = match.captured(1).remove(QChar('\'')).replace(",,", ",");
        QString result = match.captured(2);
        if (result.toInt() == 1)
        {
            auto it = std::find(answer.begin(), answer.end(), steps);
            if (answer.end() == it)
                answer.append(steps);
        }
    }

    if (!answer.isEmpty())
    {
        answer_ = answer;
        showInFront();
    }
}

bool CheatWidget::applySkin(const QString& skin)
{
    QString s;
    if (QFileInfo::exists(skin) || skin.startsWith(":/skins"))
    {
        QFileInfo fi(skin);
        if (fi.suffix() == "xml")
        {
            // load by skin configuration file - *.xml
            s = skin;
        }
        else
        {
            // load by skin package - *.cheatskin, should be decompressed first
            if (!loadSkinPackage(skin, s))
            {
                return false;
            }
        }
    }
    else
    {
        if (applySkin(QString(":/skins/%1.cheatskin").arg(skin)))
            return true;
        // load by skin name
        s = QApplication::applicationDirPath();
        const QString skinPath = QString("/skins/%1.xml").arg(skin);
#if defined(Q_OS_MAC)
        QDir d(s);
        d.cdUp();
        d.cd("Resources");
        s = d.absolutePath() + skinPath;
#else
        s += skinPath;
#endif
        if (!QFile::exists(s))
        {
            // load by skin package - *.cheatskin, should be decompressed first
            int index = s.lastIndexOf(".xml");
            Q_ASSERT(index > 0);
            s.remove(index, 4);
            s.append(".cheatskin");
            if (!loadSkinPackage(s, s))
            {
                return false;
            }
        }
    }
        
    QString imagePath;
    QString inputStyle;
    int cutTop = -1, cutBottom = -1;
    if (!loadSkinConfiguration(s, imagePath, inputStyle, cutTop, cutBottom))
    {
        return false;
    }

    if (!backgroundImage_.load(imagePath))
    {
        qCritical() << "can't load picture from " << imagePath;
        return false;
    }

    QSize size = backgroundImage_.size();

    if (cutTop >= 0 && cutBottom > cutTop)
    {
        QPixmap topPartBackgroundImage = backgroundImage_.copy(0, 0, size.width(), cutTop);
        QPixmap cutPartBackgroundImage = backgroundImage_.copy(0, cutTop, size.width(), cutBottom - cutTop);
        QPixmap bottomPartBackgroundImage = backgroundImage_.copy(0, cutBottom, size.width(), size.height() - cutBottom);
        size.setHeight(size.height() - (cutBottom - cutTop));
        qDebug() << topPartBackgroundImage.size() << cutPartBackgroundImage.size() << bottomPartBackgroundImage.size() << size << backgroundImage_.size();
        QPixmap t(size);
        t.fill(Qt::transparent);
        QPainter painter(&t);
        painter.drawPixmap(0, 0, size.width(), cutTop, topPartBackgroundImage);
        painter.drawPixmap(0, cutTop, size.width(), size.height()- cutTop, bottomPartBackgroundImage);
        backgroundImage_ = t.copy(0, 0, size.width(), size.height());
    }

    if (size.width() < widgetMinWidth_)
    {
        leftPartBackgroundImage_ = backgroundImage_.copy(0, 0,
                                                         size.width() / 2 -1, size.height());
        midPartBackgroundImage_ = backgroundImage_.copy(size.width() / 2 - 1, 0,
                                                        2, size.height()).scaled(widgetMinWidth_ - (size.width() - 2), size.height());
        rightPartBackgroundImage_ = backgroundImage_.copy(size.width() / 2 + 1, 0,
                                                          size.width() / 2 - 1, size.height());

        size.setWidth(widgetMinWidth_);
    }
    resize(1, 1);
    resize(size);
    
    return true;
}

bool CheatWidget::loadSkinConfiguration(const QString& configurationPath, QString& bgImagePath, QString& inputStyle, int& cutTop, int& cutBottom)
{
    QDomDocument doc;
    QFile file(configurationPath);
    if (!file.open(QIODevice::ReadOnly))
    {
        qCritical() << "can't open skin configuration file" << configurationPath;
        return false;
    }

    if (!doc.setContent(&file))
    {
        qCritical() << "can't parse skin configuration file" << configurationPath;
        file.close();
        return false;
    }
    file.close();

    QDomElement docElem = doc.documentElement();
    QDomElement imageElem = docElem.firstChildElement("image");
    if (imageElem.isNull())
    {
        qCritical() << "missing image element in skin configuration file" << configurationPath;
        return false;
    }
    
    QFileInfo cfg(configurationPath);

    bgImagePath = QString("%1/%2").arg(cfg.absolutePath()).arg(imageElem.text());

    QDomElement issElem = docElem.firstChildElement("inputstyle");
    if (issElem.isNull())
    {
        qCritical() << "missing inputstyle element in skin configuration file" << configurationPath;
        return false;
    }
    inputStyle = issElem.text();

    QDomElement cutTopElem = docElem.firstChildElement("cuttop");
    if (!cutTopElem.isNull())
    {
        cutTop = cutTopElem.text().toInt();
    }
    QDomElement cutBottomElem = docElem.firstChildElement("cutbottom");
    if (!cutBottomElem.isNull())
    {
        cutBottom = cutBottomElem.text().toInt();
    }

    return true;
}

bool CheatWidget::loadSkinPackage(const QString& skinPath, QString& configurationPath)
{
    QString dirName = QStandardPaths::writableLocation(QStandardPaths::AppLocalDataLocation) % "/SkinTmp";
    QDir dir(dirName);
    dir.removeRecursively();
    dir.mkpath(dirName);
    QStringList files = JlCompress::extractDir(skinPath, dirName);
    if (files.empty())
    {
        qCritical() << "extracting" << skinPath << "to" << dirName << "failed";
        return false;
    }
    configurationPath = dirName % "/skin.xml";
    if (QFile::exists(configurationPath))
        return true;

    configurationPath = dirName % "/" % QFileInfo(skinPath).completeBaseName() % ".xml";
    if (QFile::exists(configurationPath))
        return true;

    configurationPath.clear();
    qCritical() << "can't find configuration file in skin package" << skinPath;
    return false;
}
