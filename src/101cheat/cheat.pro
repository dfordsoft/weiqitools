QT       += core gui widgets xml concurrent sql

TARGET = Cheat
TEMPLATE = app
CONFIG += c++11 precompile_header
PRECOMPILED_HEADER = stdafx.h

include($$PWD/../../3rdparty/UGlobalHotkey/uglobalhotkey.pri)
include($$PWD/../../3rdparty/quazip-0.7.2/quazip.pri)
include($$PWD/../../3rdparty/qtsingleapplication/qtsingleapplication.pri)
include($$PWD/../../3rdparty/Boost.pri)

SOURCES += main.cpp\
        gui/cheatwidget.cpp 

HEADERS  += stdafx.h \
    gui/cheatwidget.h 


INCLUDEPATH += $$PWD $$PWD/gui 

CONFIG(release, debug|release) : {
    DEFINES += QT_NO_DEBUG_OUTPUT=1 QT_NO_INFO_OUTPUT=1
}

unix: !macx: {
    LIBS += -lz
}

macx: {
    ICON = cheat.icns
    icon.path = $$PWD
    icon.files += cheat.png
    INSTALLS += icon
    LIBS+=-L$$PWD/../../3rdparty/zlib-1.2.8 -lz

    CONFIG(release, debug|release) : {
    }
    QMAKE_INFO_PLIST = osxInfo.plist
}

win32: {
    QT += winextras

    LIBS+=-L$$PWD/../../3rdparty/zlib-1.2.8 \
        -lzlib -lOle32 -lVersion -lComctl32 -lGdi32
    # Windows icons
    RC_FILE = cheat.rc
    DISTFILES += cheat.rc
}

RESOURCES += \
    cheat.qrc
