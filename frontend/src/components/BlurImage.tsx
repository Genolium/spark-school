"use client";

import React, { useState } from "react";
import Image, { ImageProps } from "next/image";
import { getBlurDataURL } from "@/lib/blurData";

export interface BlurImageProps extends Omit<ImageProps, "placeholder"> {
  blurDataURL?: string;
}

/**
 * BlurImage — Progressive blur-up image loader.
 * Renders a blurred 24x16 micro-thumbnail immediately, then smoothly
 * sharpens to 100% crystal quality with a 700ms ease-out transition on load.
 */
export function BlurImage({
  src,
  alt,
  className = "",
  onLoad,
  priority,
  blurDataURL,
  ...props
}: BlurImageProps) {
  const [isLoaded, setIsLoaded] = useState(false);
  const srcString = typeof src === "string" ? src : "";
  const effectiveBlur = blurDataURL || getBlurDataURL(srcString);

  return (
    <Image
      src={src}
      alt={alt}
      priority={priority}
      placeholder="blur"
      blurDataURL={effectiveBlur}
      onLoad={(e) => {
        setIsLoaded(true);
        if (onLoad) {
          onLoad(e);
        }
      }}
      className={`transition-all duration-700 ease-out ${
        isLoaded
          ? "scale-100 blur-0 opacity-100"
          : "scale-[1.03] blur-md opacity-75"
      } ${className}`}
      {...props}
    />
  );
}
