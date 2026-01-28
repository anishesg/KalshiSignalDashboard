import MarketDetail from './MarketDetail'

interface DetailsPanelProps {
  marketTicker: string
  onClose: () => void
}

function DetailsPanel({ marketTicker, onClose }: DetailsPanelProps) {
  return <MarketDetail marketTicker={marketTicker} onClose={onClose} />
}

export default DetailsPanel

