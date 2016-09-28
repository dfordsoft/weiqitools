#include "stdafx.h"
#include "cheatwidget.h"
#include "qtsingleapplication.h"
#include <boost/scope_exit.hpp>

int main(int argc, char *argv[])
{
#if !defined(Q_OS_WIN)
    // increase the number of file that can be opened.
    struct rlimit rl;
    getrlimit(RLIMIT_NOFILE, &rl);

    rl.rlim_cur = qMin(rl.rlim_cur, rl.rlim_max);
    setrlimit(RLIMIT_NOFILE, &rl);
#endif
    SharedTools::QtSingleApplication a("101Cheat", argc, argv);

    QCoreApplication::setApplicationName("101Cheat");
    QCoreApplication::setApplicationVersion("1.0");
    QCoreApplication::setOrganizationDomain("dfordsoft.com");
    QCoreApplication::setOrganizationName("DForD Software");

    if (a.isRunning())
    {
        return 0;
    }

#if defined(Q_OS_WIN)
    CoInitialize(NULL);
    BOOST_SCOPE_EXIT(void) {
        CoUninitialize();
    } BOOST_SCOPE_EXIT_END
#endif

    CheatWidget w;
    w.show();

    QScreen* scr = a.primaryScreen();
    QSize sz = scr->availableSize();
    w.move(sz.width()/4, sz.height()/5);

    return a.exec();
}
