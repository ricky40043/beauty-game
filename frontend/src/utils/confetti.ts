import confetti from 'canvas-confetti'

/** 公布得獎時的彩帶 */
export function celebrate() {
  const shoot = (originX: number) =>
    confetti({
      particleCount: 70,
      spread: 68,
      startVelocity: 45,
      origin: { x: originX, y: 0.7 },
      colors: ['#f92e83', '#ff9bc4', '#ffd166', '#8ecae6', '#ffffff'],
    })

  shoot(0.25)
  shoot(0.75)
  setTimeout(() => shoot(0.5), 220)
}

/** 全場結算時放久一點 */
export function celebrateFinale() {
  const end = Date.now() + 1600

  const frame = () => {
    confetti({ particleCount: 4, angle: 60, spread: 60, origin: { x: 0 } })
    confetti({ particleCount: 4, angle: 120, spread: 60, origin: { x: 1 } })
    if (Date.now() < end) requestAnimationFrame(frame)
  }

  frame()
}

/** 清掉還在飄的彩帶，冠軍動畫開場前用，不然上一題的彩帶會蓋在開場畫面上 */
export function clearConfetti() {
  confetti.reset()
}
