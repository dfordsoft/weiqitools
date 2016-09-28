INCLUDEPATH += $$PWD/

unix: !macx : {
    QT += x11extras
}

SOURCES += \
    $$PWD/ukeysequence.cpp \
    $$PWD/uglobalhotkeys.cpp \
    $$PWD/uexception.cpp


