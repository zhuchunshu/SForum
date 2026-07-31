import { describe, expect, test } from 'bun:test'
import { Window } from 'happy-dom'
import { forumImageViewerSlides } from '../../app/utils/forum/forumImageViewer'

describe('forum image viewer gallery', () => {
  test('keeps each rich-content surface isolated and prefers the original link', () => {
    const window = new Window({ url: 'https://forum.test/t/1' })
    const document = window.document
    document.body.innerHTML = `
      <article data-sforum-image-gallery="topic">
        <a href="/media/attachments/a/original" data-sforum-image-viewer="1">
          <img src="/media/attachments/a" width="1600" height="900" alt="first">
        </a>
        <img src="/legacy.png" width="640" height="1280" alt="legacy">
      </article>
      <article data-sforum-image-gallery="comment-2">
        <img src="/comment.png" width="400" height="300" alt="comment">
      </article>
    `

    const target = document.querySelector('a img')!
    const result = forumImageViewerSlides(target)

    expect(result?.index).toBe(0)
    expect(result?.slides).toHaveLength(2)
    expect(result?.slides[0]).toMatchObject({
      src: 'https://forum.test/media/attachments/a/original',
      width: 1600,
      height: 900,
      alt: 'first'
    })
    expect(result?.slides[1]).toMatchObject({
      src: 'https://forum.test/legacy.png',
      width: 640,
      height: 1280,
      alt: 'legacy'
    })
  })

  test('ignores images outside a declared post or comment gallery', () => {
    const window = new Window({ url: 'https://forum.test/' })
    const image = window.document.createElement('img')
    image.src = '/logo.png'
    window.document.body.append(image)

    expect(forumImageViewerSlides(image)).toBeNull()
  })
})
