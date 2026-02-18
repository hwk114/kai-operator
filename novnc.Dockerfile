FROM novnc:20260214

ENV DEBIAN_FRONTEND=noninteractive \
    DISPLAY=:1 \
    VNC_PORT=5901 \
    VNC_RESOLUTION=1920x1080 \
    VNC_DEPTH=24 \
    LANG=zh_CN.UTF-8 \
    LC_ALL=zh_CN.UTF-8 \
    TZ=Asia/Shanghai \
    XKL_XMODMAP_DISABLE=1

WORKDIR /home/ubuntu

RUN sed -i '/zh_CN.UTF-8/s/^# //g' /etc/locale.gen \
    && locale-gen

RUN mkdir -p ~/.vnc \
    && echo '#!/bin/bash' > ~/.vnc/xstartup \
    && echo 'unset SESSION_MANAGER' >> ~/.vnc/xstartup \
    && echo 'unset DBUS_SESSION_BUS_ADDRESS' >> ~/.vnc/xstartup \
    && echo 'export XDG_CURRENT_DESKTOP=XFCE' >> ~/.vnc/xstartup \
    && echo 'export XDG_SESSION_TYPE=x11' >> ~/.vnc/xstartup \
    && echo 'export GTK_IM_MODULE=fcitx' >> ~/.vnc/xstartup \
    && echo 'export QT_IM_MODULE=fcitx' >> ~/.vnc/xstartup \
    && echo 'export XMODIFIERS=@im=fcitx' >> ~/.vnc/xstartup \
    && echo 'export IMSETTINGS_INTEGRATE_DESKTOP=yes' >> ~/.vnc/xstartup \
    && echo 'fcitx5 -d &' >> ~/.vnc/xstartup \
    && echo 'xfsettingsd &' >> ~/.vnc/xstartup \
    && echo 'xfce4-session &' >> ~/.vnc/xstartup \
    && echo 'startxfce4' >> ~/.vnc/xstartup \
    && chmod +x ~/.vnc/xstartup \
    && mkdir -p /home/ubuntu/.vnc \
    && cp ~/.vnc/xstartup /home/ubuntu/.vnc/xstartup \
    && chown -R ubuntu:ubuntu /home/ubuntu/.vnc

RUN echo '#!/bin/bash\n\
tigervncserver :1 -geometry ${VNC_RESOLUTION} -depth ${VNC_DEPTH} -SecurityTypes None -localhost no -I-KNOW-THIS-IS-INSECURE\n\
sleep 2\n\
websockify --web=/usr/share/novnc 6081 0.0.0.0:5901\n\
wait\n' > /start.sh && chmod +x /start.sh

RUN echo '<!DOCTYPE html>\
<html>\
<head>\
<meta charset="utf-8">\
<title>noVNC</title>\
<script>\
var host = window.location.hostname;\
var port = window.location.port;\
if (!port) { port = window.location.protocol === "https:" ? "443" : "80"; }\
var path = window.location.pathname.replace(/^\//, "").replace(/\/vnc.*$/, "").replace(/\/$/, "") + "/websockify";\
window.location.href = "vnc_auto.html#host=" + encodeURIComponent(host) + "&port=" + encodeURIComponent(port) + "&path=" + encodeURIComponent(path);\
</script>\
</head>\
<body><p>Redirecting to noVNC...</p></body>\
</html>' > /usr/share/novnc/index.html

USER ubuntu

EXPOSE 6081 5901

CMD ["/start.sh"]
