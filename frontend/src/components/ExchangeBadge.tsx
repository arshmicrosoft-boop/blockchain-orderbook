import type { SVGProps } from 'react';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { Badge } from '@/components/ui/badge';
import asterLogo from '@/assets/aster.png';
import bingxLogo from '@/assets/bingx.png';
import hyperliquidLogo from '@/assets/hyperliquid.png';

// SVG Icon Components
function TokenBrandedBybit(props: SVGProps<SVGSVGElement>) {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" width="1em" height="1em" viewBox="0 0 24 24" {...props}>{/* Icon from Web3 Icons Branded by 0xa3k5 - https://github.com/0xa3k5/web3icons/blob/main/LICENCE */}<g fill="none"><path fill="#F6A500" d="M15.829 13.626V9h.93v4.626z" /><path fill="#fff" d="M4.993 15H3v-4.626h1.913c.93 0 1.471.507 1.471 1.3c0 .513-.348.845-.588.955c.287.13.655.423.655 1.04c0 .863-.609 1.33-1.458 1.33m-.154-3.82h-.91v1.065h.91c.395 0 .615-.214.615-.533c0-.317-.22-.532-.615-.532m.06 1.877h-.97v1.137h.97c.42 0 .622-.259.622-.571s-.201-.565-.622-.565zm4.388.046V15h-.923v-1.898l-1.431-2.728h1.01l.889 1.864l.877-1.864h1.01zM13.355 15h-1.993v-4.626h1.913c.93 0 1.47.507 1.47 1.3c0 .513-.347.845-.588.955c.287.13.655.423.655 1.04c0 .863-.608 1.33-1.457 1.33m-.155-3.82h-.91v1.065h.91c.395 0 .616-.214.616-.533c0-.317-.22-.532-.616-.532m.06 1.877h-.97v1.137h.97c.422 0 .622-.259.622-.571s-.2-.565-.622-.565zm6.495-1.876V15h-.929v-3.82h-1.245v-.806H21v.806z" /></g></svg>
  );
}

function SimpleIconsOkx(props: SVGProps<SVGSVGElement>) {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" width="1em" height="1em" viewBox="0 0 24 24" {...props}>{/* Icon from Simple Icons by Simple Icons Collaborators - https://github.com/simple-icons/simple-icons/blob/develop/LICENSE.md */}<path fill="currentColor" d="M7.15 8.685c.03.02.05.06.05.1v6.44c0 .04-.02.08-.05.1a.17.17 0 0 1-.11.05H.16a.17.17 0 0 1-.11-.05a.14.14 0 0 1-.05-.1v-6.44c0-.04.02-.08.05-.1a.17.17 0 0 1 .11-.05h6.88c.04 0 .08.02.11.05m-2.35 2.35a.14.14 0 0 0-.05-.1a.16.16 0 0 0-.11-.05H2.56a.17.17 0 0 0-.11.04a.14.14 0 0 0-.05.1v1.95c0 .04.02.08.05.1c.03.04.07.05.11.05h2.08c.04 0 .08-.01.11-.04a.14.14 0 0 0 .05-.1zm16.8 0v1.94c0 .09-.07.15-.16.15h-2.08c-.09 0-.16-.06-.16-.15v-1.94c0-.08.07-.15.16-.15h2.08c.09 0 .16.06.16.15m-2.4-2.25v1.95c0 .08-.07.15-.16.15h-2.08c-.09 0-.16-.07-.16-.15v-1.95c0-.08.07-.15.16-.15h2.08c.09 0 .16.07.16.15m4.8 0v1.95c0 .08-.07.15-.16.15h-2.08c-.09 0-.16-.07-.16-.15v-1.95c0-.08.07-.15.16-.15h2.08c.09 0 .16.07.16.15m-4.8 4.5v1.94c0 .08-.07.15-.16.15h-2.08c-.09 0-.16-.07-.16-.15v-1.95c0-.08.07-.15.16-.15h2.08c.09 0 .16.07.16.15zm4.8 0v1.94c0 .08-.07.15-.16.15h-2.08c-.09 0-.16-.07-.16-.15v-1.95c0-.08.07-.15.16-.15h2.08c.09 0 .16.07.16.15zm-8.4-4.5v1.95c0 .08-.07.15-.16.15h-2.08c-.09 0-.16-.07-.16-.15v-1.95c0-.08.07-.15.16-.15h2.08c.09 0 .16.07.16.15m0 4.5v1.94c0 .08-.07.15-.16.15h-2.08c-.09 0-.16-.07-.16-.15v-1.95c0-.08.07-.15.16-.15h2.08c.09 0 .16.07.16.15zm-2.4-.3a.16.16 0 0 1-.16.15H10.8v2.08c0 .04-.02.08-.05.11a.17.17 0 0 1-.11.04H8.56a.17.17 0 0 1-.12-.04a.14.14 0 0 1-.04-.1v-6.45c0-.04.01-.08.04-.1a.17.17 0 0 1 .12-.05h2.08c.04 0 .08.02.11.05c.03.02.05.06.05.1v2.1h2.24c.04 0 .08.01.11.04s.05.07.05.1z" /></svg>
  );
}

function TokenBrandedKraken(props: SVGProps<SVGSVGElement>) {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" width="1em" height="1em" viewBox="0 0 24 24" {...props}>{/* Icon from Web3 Icons Branded by 0xa3k5 - https://github.com/0xa3k5/web3icons/blob/main/LICENCE */}<path fill="#7133F5" d="M11.998 4.5C7.028 4.5 3 8.774 3 14.047v4.09c0 .753.575 1.363 1.285 1.363s1.288-.61 1.288-1.362v-4.091c0-.755.573-1.365 1.285-1.365c.71 0 1.284.61 1.284 1.365v4.09c0 .753.575 1.363 1.285 1.363c.712 0 1.286-.61 1.286-1.362v-4.091c0-.755.575-1.365 1.285-1.365c.712 0 1.289.61 1.289 1.365v4.09c0 .753.574 1.363 1.284 1.363s1.285-.61 1.285-1.362v-4.091c0-.755.574-1.365 1.288-1.365c.71 0 1.285.61 1.285 1.365v4.09c0 .753.575 1.363 1.287 1.363c.71 0 1.284-.61 1.284-1.362v-4.091C21 8.774 16.97 4.5 11.998 4.5" /></svg>
  );
}

function TokenBrandedBinance(props: SVGProps<SVGSVGElement>) {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" width="1em" height="1em" viewBox="0 0 24 24" {...props}>{/* Icon from Web3 Icons Branded by 0xa3k5 - https://github.com/0xa3k5/web3icons/blob/main/LICENCE */}<path fill="#F0B90B" d="m7.068 12l-2.03 2.03L3.003 12l2.03-2.03zm4.935-4.935l3.482 3.483l2.03-2.03L12.003 3L6.485 8.518l2.03 2.03zm6.964 2.905L16.937 12l2.03 2.03l2.03-2.03zm-6.964 6.965L8.52 13.452l-2.03 2.03L12.003 21l5.512-5.518l-2.03-2.03zm0-2.905l2.03-2.03l-2.03-2.03L9.967 12z" /></svg>
  );
}

function TokenBrandedCoinbase(props: SVGProps<SVGSVGElement>) {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" width="1em" height="1em" viewBox="0 0 24 24" {...props}>{/* Icon from Web3 Icons Branded by 0xa3k5 - https://github.com/0xa3k5/web3icons/blob/main/LICENCE */}<g fill="none"><path fill="#0E5BFF" d="M3 12a9 9 0 1 1 18 0a9 9 0 0 1-18 0" /><path fill="#fff" fillRule="evenodd" d="M12 18.375a6.375 6.375 0 1 0 0-12.75a6.375 6.375 0 0 0 0 12.75m-.75-8.25c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125h1.5c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125z" clipRule="evenodd" /></g></svg>
  );
}

type ExchangeIconProps = SVGProps<SVGSVGElement> & {
  exchange: string;
};

/**
 * Renders the appropriate icon for an exchange
 */
function ExchangeIcon({ exchange, className, ...props }: ExchangeIconProps) {
  const exchangeLower = exchange.toLowerCase().replace(/f$/, ''); // Remove 'f' suffix for matching

  // Check for PNG logos first
  if (exchangeLower.includes('aster')) {
    return <img src={asterLogo} alt="Aster" className={className} />;
  }
  if (exchangeLower.includes('bingx')) {
    return <img src={bingxLogo} alt="BingX" className={className} />;
  }
  if (exchangeLower.includes('hyperliquid')) {
    return <img src={hyperliquidLogo} alt="Hyperliquid" className={className} />;
  }

  // SVG icons
  if (exchangeLower.includes('bybit')) {
    return <TokenBrandedBybit className={className} {...props} />;
  }
  if (exchangeLower.includes('okx')) {
    return <SimpleIconsOkx className={className} {...props} />;
  }
  if (exchangeLower.includes('kraken')) {
    return <TokenBrandedKraken className={className} {...props} />;
  }
  if (exchangeLower.includes('binance')) {
    return <TokenBrandedBinance className={className} {...props} />;
  }
  if (exchangeLower.includes('coinbase')) {
    return <TokenBrandedCoinbase className={className} {...props} />;
  }

  // Fallback: return first letter
  return (
    <span className={`font-bold text-sm inline-flex items-center justify-center ${className || ''}`}>
      {exchange.charAt(0).toUpperCase()}
    </span>
  );
}

type ExchangeBadgeProps = {
  exchange: string;
  showLabel?: boolean;
  showMarketType?: boolean;
  className?: string;
  iconClassName?: string;
};

/**
 * Determines if an exchange is perpetual futures based on naming convention
 */
function isPerps(exchange: string): boolean {
  return exchange.endsWith('f');
}

/**
 * Gets the clean exchange name without the 'f' suffix
 */
function getCleanExchangeName(exchange: string): string {
  return exchange.replace(/f$/, '');
}

/**
 * Gets the market type label
 */
function getMarketType(exchange: string): string {
  return isPerps(exchange) ? 'Perps' : 'Spot';
}

/**
 * Comprehensive exchange badge component with icon, market type badge, and tooltip
 * Can be used throughout the application for consistent exchange representation
 */
export function ExchangeBadge({
  exchange,
  showLabel = false,
  showMarketType = true,
  className = '',
  iconClassName = 'w-4 h-4',
}: ExchangeBadgeProps) {
  const cleanName = getCleanExchangeName(exchange);
  const marketType = getMarketType(exchange);
  const isPerpetual = isPerps(exchange);

  const content = (
    <div className={`inline-flex items-center gap-1.5 ${className}`}>
      <ExchangeIcon exchange={exchange} className={iconClassName} />
      {showLabel && (
        <span className="capitalize">{cleanName}</span>
      )}
      {showMarketType && (
        <Badge
          variant={isPerpetual ? 'default' : 'secondary'}
          className="text-[9px] px-1 py-0 h-4 font-medium"
        >
          {marketType}
        </Badge>
      )}
    </div>
  );

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <div className="inline-flex cursor-help">
          {content}
        </div>
      </TooltipTrigger>
      <TooltipContent className="bg-popover text-popover-foreground border border-border">
        <div className="text-xs space-y-0.5">
          <div className="font-semibold capitalize">{cleanName}</div>
          <div className="text-xs opacity-70">
            {isPerpetual ? 'Perpetual Futures' : 'Spot Market'}
          </div>
        </div>
      </TooltipContent>
    </Tooltip>
  );
}