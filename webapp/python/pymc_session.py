from datetime import timedelta
import json
import os

from flask.sessions import SessionInterface, SessionMixin
from werkzeug.datastructures import CallbackDict


class Session(CallbackDict, SessionMixin):

    def __init__(self, initial=None, sid=None, new=False):
        def on_update(self):
            self.modified = True
        CallbackDict.__init__(self, initial, on_update)
        self.sid = sid
        self.new = new
        self.modified = False


class SessionInterface(SessionInterface):

    def __init__(self, memcache, prefix='session:'):
        self.memcache = memcache
        self.prefix = prefix

    def generate_sid(self):
        return os.urandom(8).hex()

    def get_memcache_expiration_time(self, app, session):
        if session.permanent:
            return app.permanent_session_lifetime
        return timedelta(days=1)

    def open_session(self, app, request):
        sid = request.cookies.get(app.config["SESSION_COOKIE_NAME"])
        if not sid:
            sid = self.generate_sid()
            return Session(sid=sid, new=True)
        val = self.memcache.get(self.prefix + sid)
        if val is not None:
            data = json.loads(val.decode('utf-8'))
            return Session(data, sid=sid)
        return Session(sid=sid, new=True)

    def save_session(self, app, session, response):
        domain = self.get_cookie_domain(app)
        if not session:
            self.memcache.delete(self.prefix + session.sid)
            if session.modified:
                response.delete_cookie(app.config["SESSION_COOKIE_NAME"],
                                       domain=domain)
            return
        memcache_exp = self.get_memcache_expiration_time(app, session)
        cookie_exp = self.get_expiration_time(app, session)
        val = json.dumps(dict(session))
        self.memcache.set(self.prefix + session.sid, val,
                          int(memcache_exp.total_seconds()))
        response.set_cookie(app.config["SESSION_COOKIE_NAME"], session.sid,
                            expires=cookie_exp, httponly=True,
                            domain=domain)
