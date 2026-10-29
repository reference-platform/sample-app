FROM python:3.15.2-slim-trixie AS build
RUN python -m venv /venv
COPY requirements.txt .
RUN /venv/bin/pip install --no-cache-dir -r requirements.txt

FROM python:3.15.2-slim-trixie
RUN useradd --system --uid 10001 --no-create-home app
COPY --from=build /venv /venv
WORKDIR /srv
COPY app ./app
ENV PATH=/venv/bin:$PATH \
    PYTHONUNBUFFERED=1 \
    PORT=8000
USER 10001
EXPOSE 8000
CMD ["python", "-m", "app"]
